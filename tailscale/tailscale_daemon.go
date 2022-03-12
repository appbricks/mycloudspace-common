package tailscale

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-multierror/multierror"
	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"golang.zx2c4.com/wireguard/device"
	"inet.af/netaddr"
	"tailscale.com/envknob"
	"tailscale.com/ipn"
	"tailscale.com/ipn/ipnserver"
	"tailscale.com/logpolicy"
	"tailscale.com/logtail"
	"tailscale.com/net/dns"
	"tailscale.com/net/netns"
	"tailscale.com/net/proxymux"
	"tailscale.com/net/socks5"
	"tailscale.com/net/tsdial"

	"tailscale.com/net/tstun"
	"tailscale.com/paths"
	"tailscale.com/safesocket"
	"tailscale.com/types/logger"
	"tailscale.com/util/clientmetric"
	"tailscale.com/version/distro"
	"tailscale.com/wgengine"
	"tailscale.com/wgengine/monitor"
	"tailscale.com/wgengine/netstack"
	"tailscale.com/wgengine/router"

	cb_logger "github.com/mevansam/goutils/logger"
)

type TailscaleDaemon struct {
	// writer to which all tailscale
	// logs will be written. this can 
	// be intercepted and interpretted
	// or re-routed, etc.
	logOut io.Writer

	// tunname is a /dev/net/tun tunnel name ("tailscale0"), the
	// string "userspace-networking", "tap:TAPNAME[:BRIDGENAME]"
	// or comma-separated list thereof.
	tunname string
	
	port           uint16
	statePath      string
	socketPath     string
	birdSocketPath string	
	socksAddr      string
	httpProxyAddr  string

	// log verbosity level; 0 is default, 1 or higher are increasingly verbose
	verbose int
	// listen address ([ip]:port) of optional debug server
	Debug string

	// tunnel device
	devName string
	// wireguard control service
	wgDevice *device.Device
	
	// tailscale services cancel func
	cancel context.CancelFunc

	// released when ipn server exits
	exit *sync.WaitGroup
}

func NewTailscaleDaemon(statePath string, logOut io.Writer) *TailscaleDaemon {

	var (
		socketPath string

		verboseLevel int
	)
	
	// remove stale config socket if found (*nix systems only)
	if socketPath = paths.DefaultTailscaledSocket(); len(socketPath) > 0 {
		os.Remove(socketPath)
	}

	switch logrus.GetLevel() {
	case logrus.TraceLevel:
		fallthrough
	case logrus.DebugLevel:
		verboseLevel = 2
	default:
		verboseLevel = 0
	}

	return &TailscaleDaemon{
		logOut: logOut,

		// tunnel interface name
		tunname: defaultTunName(),
		// UDP port to listen on for WireGuard and 
		// peer-to-peer traffic; 0 means automatically 
		// select
		port: 0,
		// "path of state file
		statePath: statePath,
		// path of the service unix socket
		socketPath: paths.DefaultTailscaledSocket(),
		// path of the bird unix socket
		birdSocketPath: "",
		// optional [ip]:port to run a SOCK5 server (e.g. "localhost:1080")
		socksAddr: "",

		verbose: verboseLevel,

		exit: &sync.WaitGroup{},
	}
}

func (tsd *TailscaleDaemon) TunnelDeviceName() string {
	return tsd.devName
}

func (tsd *TailscaleDaemon) Start() error {
	return tsd.run()
}

func (tsd *TailscaleDaemon) Stop() {
	tsd.cancel()
	cb_logger.TraceMessage("TailscaleDaemon.Stop(): Waiting for tailscale daemon services to stop")
	tsd.exit.Wait()
}

func (tsd *TailscaleDaemon) Cleanup() {
	dns.Cleanup(log.Printf, tsd.tunname)
	router.Cleanup(log.Printf, tsd.tunname)
}

// copied from tailscale/cmd/tailscaled

// defaultTunName returns the default tun device name for the platform.
func defaultTunName() string {
	switch runtime.GOOS {
	case "openbsd":
		return "tun"
	case "windows":
		return "Tailscale"
	case "darwin":
		// "utun" is recognized by wireguard-go/tun/tun_darwin.go
		// as a magic value that uses/creates any free number.
		return "utun"
	case "linux":
		if distro.Get() == distro.Synology {
			// Try TUN, but fall back to userspace networking if needed.
			// See https://github.com/tailscale/tailscale-synology/issues/35
			return "tailscale0,userspace-networking"
		}
	}
	return "tailscale0"
}

func (tsd *TailscaleDaemon) run() error {
	
	var (
		err error
		ctx context.Context

		logf logger.Logf
	)

	if tsd.statePath == "" {
		log.Fatalf("state path is required")
	}

	logpolicy.MyCSLogOut = tsd.logOut

	pol := logpolicy.New(logtail.CollectionNode)
	pol.SetVerbosityLevel(tsd.verbose)
	defer func() {
		// Finish uploading logs after closing everything else.
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = pol.Shutdown(ctx)
	}()

	logf = cb_logger.DebugMessage
	if envknob.Bool("TS_DEBUG_MEMORY") {
		logf = logger.RusagePrefixLog(logf)
	}
	logf = logger.RateLimitedFn(logf, 5*time.Second, 5, 100)

	var debugMux *http.ServeMux
	if tsd.Debug != "" {
		debugMux = newDebugMux()
	}

	linkMon, err := monitor.New(logf)
	if err != nil {
		return fmt.Errorf("monitor.New: %w", err)
	}
	pol.Logtail.SetLinkMonitor(linkMon)

	socksListener, httpProxyListener := mustStartProxyListeners(tsd.socksAddr, tsd.httpProxyAddr)

	dialer := new(tsdial.Dialer) // mutated below (before used)
	e, useNetstack, err := tsd.createEngine(logf, linkMon, dialer)
	if err != nil {
		return fmt.Errorf("createEngine: %w", err)
	}
	if _, ok := e.(wgengine.ResolvingEngine).GetResolver(); !ok {
		panic("internal error: exit node resolver not wired up")
	}
	if debugMux != nil {
		if ig, ok := e.(wgengine.InternalsGetter); ok {
			if _, mc, ok := ig.GetInternals(); ok {
				debugMux.HandleFunc("/debug/magicsock", mc.ServeHTTPDebug)
			}
		}
		go runDebugServer(debugMux, tsd.Debug)
	}

	ns, err := newNetstack(logf, dialer, e)
	if err != nil {
		return fmt.Errorf("newNetstack: %w", err)
	}
	ns.ProcessLocalIPs = useNetstack
	ns.ProcessSubnets = useNetstack || wrapNetstack

	if useNetstack {
		dialer.UseNetstackForIP = func(ip netaddr.IP) bool {
			_, ok := e.PeerForIP(ip)
			return ok
		}
		dialer.NetstackDialTCP = func(ctx context.Context, dst netaddr.IPPort) (net.Conn, error) {
			return ns.DialContextTCP(ctx, dst)
		}
	}
	if socksListener != nil || httpProxyListener != nil {
		if httpProxyListener != nil {
			hs := &http.Server{Handler: httpProxyHandler(dialer.UserDial)}
			go func() {
				log.Fatalf("HTTP proxy exited: %v", hs.Serve(httpProxyListener))
			}()
		}
		if socksListener != nil {
			ss := &socks5.Server{
				Logf:   logger.WithPrefix(logf, "socks5: "),
				Dialer: dialer.UserDial,
			}
			go func() {
				log.Fatalf("SOCKS5 server exited: %v", ss.Serve(socksListener))
			}()
		}
	}

	e = wgengine.NewWatchdog(e)
	ctx, tsd.cancel = context.WithCancel(context.Background())

	opts := tsd.ipnServerOpts()

	store, err := ipnserver.StateStore(filepath.Join(tsd.statePath, "tailscaled.state"), logf)
	if err != nil {
		return fmt.Errorf("ipnserver.StateStore: %w", err)
	}
	srv, err := ipnserver.New(logf, pol.PublicID.String(), store, e, dialer, nil, opts)
	if err != nil {
		return fmt.Errorf("ipnserver.New: %w", err)
	}
	ns.SetLocalBackend(srv.LocalBackend())
	if err := ns.Start(); err != nil {
		log.Fatalf("failed to start netstack: %v", err)
	}
	
	if debugMux != nil {
		debugMux.HandleFunc("/debug/ipn", srv.ServeHTMLStatus)
	}

	ln, _, err := safesocket.Listen(tsd.socketPath, safesocket.WindowsLocalPort)
	if err != nil {
		return fmt.Errorf("safesocket.Listen: %v", err)
	}

	tsd.exit.Add(1)
	go func() {
		err = srv.Run(ctx, ln)
		// Cancelation is not an error: it is the only way to stop ipnserver.
		if err != nil && err != context.Canceled {
			logf("ipnserver.Run: %v", err)
		}
	
		cb_logger.TraceMessage("TailscaleDaemon.run(): Tailscale daemon services stopped")
		tsd.exit.Done()
	}()

	return nil
}

func (tsd *TailscaleDaemon) ipnServerOpts() (o ipnserver.Options) {
	// Allow changing the OS-specific IPN behavior for tests
	// so we can e.g. test Windows-specific behaviors on Linux.
	goos := os.Getenv("TS_DEBUG_TAILSCALED_IPN_GOOS")
	if goos == "" {
		goos = runtime.GOOS
	}

	o.VarRoot = tsd.statePath

	// If an absolute --state is provided try to derive
	// a state directory.
	if o.VarRoot == "" {
		home, _ := homedir.Dir()
		o.VarRoot = filepath.Join(home, ".tailscale")
	}

	switch goos {
	default:
		o.SurviveDisconnects = true
		o.AutostartStateKey = ipn.GlobalDaemonStateKey
	case "windows":
		// Not those.
	}
	return o
}

func  (tsd *TailscaleDaemon) createEngine(logf logger.Logf, linkMon *monitor.Mon, dialer *tsdial.Dialer) (e wgengine.Engine, useNetstack bool, err error) {
	if tsd.tunname == "" {
		return nil, false, errors.New("no --tun value specified")
	}
	var errs []error
	for _, name := range strings.Split(tsd.tunname, ",") {
		logf("wgengine.NewUserspaceEngine(tun %q) ...", name)
		e, useNetstack, err = tsd.tryEngine(logf, linkMon, dialer, name)
		if err == nil {
			return e, useNetstack, nil
		}
		logf("wgengine.NewUserspaceEngine(tun %q) error: %v", name, err)
		errs = append(errs, err)
	}
	return nil, false, multierror.New(errs)
}

func  (tsd *TailscaleDaemon) tryEngine(logf logger.Logf, linkMon *monitor.Mon, dialer *tsdial.Dialer, name string) (e wgengine.Engine, useNetstack bool, err error) {
	conf := wgengine.Config{
		ListenPort:  tsd.port,
		LinkMonitor: linkMon,
		Dialer:      dialer,
	}

	useNetstack = name == "userspace-networking"
	netns.SetEnabled(!useNetstack)

	if !useNetstack {
		dev, devName, err := tstun.New(logf, name)
		if err != nil {
			tstun.Diagnose(logf, name)
			return nil, false, fmt.Errorf("tstun.New(%q): %w", name, err)
		}
		conf.Tun = dev
		if strings.HasPrefix(name, "tap:") {
			conf.IsTAP = true
			e, err := wgengine.NewUserspaceEngine(logf, conf)
			return e, false, err
		}

		r, err := router.New(logf, dev, linkMon)
		if err != nil {
			dev.Close()
			return nil, false, fmt.Errorf("creating router: %w", err)
		}
		d, err := dns.NewOSConfigurator(logf, devName)
		if err != nil {
			return nil, false, fmt.Errorf("dns.NewOSConfigurator: %w", err)
		}
		conf.DNS = d
		conf.Router = r
		if wrapNetstack {
			conf.Router = netstack.NewSubnetRouterWrapper(conf.Router)
		}

		// MyCS: save tunnel device name
		tsd.devName = devName
	}
	e, err = wgengine.NewUserspaceEngine(logf, conf)
	if err != nil {
		return nil, useNetstack, err
	}

	// MyCS: save underlying wireguard device
	tsd.wgDevice = wgengine.GetWireguardDevice(e)
	
	return e, useNetstack, nil
}

var wrapNetstack = shouldWrapNetstack()

func shouldWrapNetstack() bool {
	if e := os.Getenv("TS_DEBUG_WRAP_NETSTACK"); e != "" {
		v, err := strconv.ParseBool(e)
		if err != nil {
			log.Fatalf("invalid TS_DEBUG_WRAP_NETSTACK value: %v", err)
		}
		return v
	}
	if distro.Get() == distro.Synology {
		return true
	}
	switch runtime.GOOS {
	case "windows", "darwin", "freebsd":
		// Enable on Windows and tailscaled-on-macOS (this doesn't
		// affect the GUI clients), and on FreeBSD.
		return true
	}
	return false
}

func newDebugMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/metrics", servePrometheusMetrics)
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	return mux
}

func servePrometheusMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	clientmetric.WritePrometheusExpositionFormat(w)
}

func runDebugServer(mux *http.ServeMux, addr string) {
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func newNetstack(logf logger.Logf, dialer *tsdial.Dialer, e wgengine.Engine) (*netstack.Impl, error) {
	tunDev, magicConn, ok := e.(wgengine.InternalsGetter).GetInternals()
	if !ok {
		return nil, fmt.Errorf("%T is not a wgengine.InternalsGetter", e)
	}
	return netstack.Create(logf, tunDev, e, magicConn, dialer)
}

// mustStartProxyListeners creates listeners for local SOCKS and HTTP
// proxies, if the respective addresses are not empty. socksAddr and
// httpAddr can be the same, in which case socksListener will receive
// connections that look like they're speaking SOCKS and httpListener
// will receive everything else.
//
// socksListener and httpListener can be nil, if their respective
// addrs are empty.
func mustStartProxyListeners(socksAddr, httpAddr string) (socksListener, httpListener net.Listener) {
	if socksAddr == httpAddr && socksAddr != "" && !strings.HasSuffix(socksAddr, ":0") {
		ln, err := net.Listen("tcp", socksAddr)
		if err != nil {
			log.Fatalf("proxy listener: %v", err)
		}
		return proxymux.SplitSOCKSAndHTTP(ln)
	}

	var err error
	if socksAddr != "" {
		socksListener, err = net.Listen("tcp", socksAddr)
		if err != nil {
			log.Fatalf("SOCKS5 listener: %v", err)
		}
		if strings.HasSuffix(socksAddr, ":0") {
			// Log kernel-selected port number so integration tests
			// can find it portably.
			log.Printf("SOCKS5 listening on %v", socksListener.Addr())
		}
	}
	if httpAddr != "" {
		httpListener, err = net.Listen("tcp", httpAddr)
		if err != nil {
			log.Fatalf("HTTP proxy listener: %v", err)
		}
		if strings.HasSuffix(httpAddr, ":0") {
			// Log kernel-selected port number so integration tests
			// can find it portably.
			log.Printf("HTTP proxy listening on %v", httpListener.Addr())
		}
	}

	return socksListener, httpListener
}
