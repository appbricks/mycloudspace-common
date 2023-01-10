package tailscale

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/netip"
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
	"golang.zx2c4.com/wireguard/tun"
	"tailscale.com/client/tailscale"
	"tailscale.com/envknob"
	"tailscale.com/ipn"
	"tailscale.com/ipn/ipnserver"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/ipn/store"
	"tailscale.com/logpolicy"
	"tailscale.com/logtail"
	"tailscale.com/net/dns"
	"tailscale.com/net/netns"
	"tailscale.com/net/tsdial"
	"tailscale.com/tailcfg"

	"tailscale.com/net/tstun"
	"tailscale.com/paths"
	"tailscale.com/safesocket"
	"tailscale.com/types/logger"
	"tailscale.com/version/distro"
	"tailscale.com/wgengine"
	"tailscale.com/wgengine/monitor"
	"tailscale.com/wgengine/netstack"
	"tailscale.com/wgengine/router"

	cb_logger "github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/utils"
)

type TailscaleDaemon struct {
	// tunname is a /dev/net/tun tunnel name ("tailscale0"), the
	// string "userspace-networking", "tap:TAPNAME[:BRIDGENAME]"
	// or comma-separated list thereof.
	tunname string
	
	port           uint16
	statePath      string
	socketPath     string

	// log verbosity level; 0 is default, 1 or higher are increasingly verbose
	verbose int

	// tunnel device
	devName string
	// wireguard control service
	wgDevice *device.Device

	// tailscale services context
	ctx    context.Context
	cancel context.CancelFunc

	// timer ping mesh nodes 
	// to ensure connectivitity
	nodeCheckTimer *utils.ExecTimer
	
	// released when ipn server exits
	exit *sync.WaitGroup

	// used for mutually exclusive access to internals
	mx sync.Mutex
}

var tslocalClient tailscale.LocalClient

const nodeCheckTimeout = 5000 // milliseconds

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

	// writer to which all tailscale
	// logs will be written. this can 
	// be intercepted and interpretted
	// or re-routed, etc.
	logpolicy.MyCSLogOut = logOut

	tsd := &TailscaleDaemon{
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

		verbose: verboseLevel,

		exit: &sync.WaitGroup{},
	}

	tsd.ctx, tsd.cancel = context.WithCancel(context.Background())
	tsd.nodeCheckTimer = utils.NewExecTimer(tsd.ctx, tsd.nodeCheck, false)

	return tsd
}

func (tsd *TailscaleDaemon) TunnelDeviceName() string {
	return tsd.devName
}

func (tsd *TailscaleDaemon) Start() error {
	
	// start node check timer
	if err := tsd.nodeCheckTimer.Start(nodeCheckTimeout); err != nil {
		cb_logger.ErrorMessage(
			"TailscaleDaemon.Start(): Failed to start node check timer: %s", 
			err.Error(),
		)
		return err
	}

	return tsd.run()
}

func (tsd *TailscaleDaemon) Stop() {

	// stopnode check time
	if err := tsd.nodeCheckTimer.Stop(); err != nil {
		cb_logger.ErrorMessage(
			"TailscaleDaemon.Stop(): Node check timer stopped with err: %s", 
			err.Error())	
	}	

	tsd.cancel()
	cb_logger.TraceMessage("TailscaleDaemon.Stop(): Waiting for tailscale daemon services to stop")
	tsd.exit.Wait()
}

func (tsd *TailscaleDaemon) Cleanup() {
	dns.Cleanup(log.Printf, tsd.tunname)
	router.Cleanup(log.Printf, tsd.tunname)
}

func (tsd *TailscaleDaemon) nodeCheck() (time.Duration, error) {

	var (
		err error

		status *ipnstate.Status
	)

	if status, err = tslocalClient.Status(tsd.ctx); err != nil {
		cb_logger.ErrorMessage(
			"TailscaleDaemon.nodeCheck(): Error retrieving tailscale status: %s", 
			err.Error())		

	} else {
		for _, ps := range status.Peer {

			if ps.Online {
				peerStatus := ps
				go func() {
	
					var (
						err error
						ok  bool
	
						resolvedIPs []net.IP
						ip          netip.Addr
	
						pingResult *ipnstate.PingResult
					)
	
					if resolvedIPs, err = net.LookupIP(peerStatus.DNSName); err != nil || len(resolvedIPs) == 0 {
						cb_logger.ErrorMessage(
							"TailscaleDaemon.nodeCheck(): Cannot resolve IP for node '%s'.", 
							peerStatus.DNSName,
						)
						return
					}
					if ip, ok = netip.AddrFromSlice(resolvedIPs[0]); !ok {
						cb_logger.ErrorMessage(
							"TailscaleDaemon.nodeCheck(): Invalid IP '%s' for for node '%s': %s", 
							resolvedIPs[0].String(), peerStatus.DNSName, err.Error(),
						)
						return
					}
	
					ctx, cancel := context.WithTimeout(tsd.ctx, time.Second)
					defer cancel()

					cb_logger.TraceMessage(
						"TailscaleDaemon.nodeCheck(): Pinging '%s/%s'.", 
						peerStatus.DNSName, ip.String(), 
					)
					for {
						if pingResult, err = tslocalClient.Ping(ctx, ip, tailcfg.PingDisco); 
							err != nil && !errors.Is(err, context.DeadlineExceeded) {
							
							cb_logger.DebugMessage(
								"TailscaleDaemon.nodeCheck(): Cannot reach node '%s/%s', got error: %s", 
								peerStatus.DNSName, resolvedIPs[0].String(), err.Error(),
							)	
							continue
						}
						break
					}
					if errors.Is(err, context.DeadlineExceeded) {
						cb_logger.TraceMessage(
							"TailscaleDaemon.nodeCheck(): Timed out pinging '%s/%s.", 
							peerStatus.DNSName, ip.String(),
						)	
					} else if err == nil {
						cb_logger.TraceMessage(
							"TailscaleDaemon.nodeCheck(): Pong from '%s/%s' at endpoint %s.", 
							pingResult.NodeName, pingResult.IP, pingResult.Endpoint,
						)	
					}
				}()
			}
		}
	}

	return nodeCheckTimeout, nil
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

		logf logger.Logf
	)

	if tsd.statePath == "" {
		return fmt.Errorf("state path is required")
	}
	
	// set up tailscale daemon logging

	pol := logpolicy.New(logtail.CollectionNode)
	pol.SetVerbosityLevel(tsd.verbose)
	defer func() {
		// Finish uploading logs after closing everything else.
		ctx, cancel := context.WithTimeout(tsd.ctx, time.Second)
		defer cancel()
		_ = pol.Shutdown(ctx)
	}()

	logf = cb_logger.DebugMessage
	if envknob.Bool("TS_DEBUG_MEMORY") {
		logf = logger.RusagePrefixLog(logf)
	}
	logf = logger.RateLimitedFn(logf, 5*time.Second, 5, 100)

	linkMon, err := monitor.New(logf)
	if err != nil {
		return fmt.Errorf("monitor.New: %w", err)
	}
	pol.Logtail.SetLinkMonitor(linkMon)

	// set up tailscale daemon engine

	dialer := new(tsdial.Dialer) // mutated below (before used)
	engine, useNetstack, err := tsd.createEngine(logf, linkMon, dialer)
	if err != nil {
		return fmt.Errorf("createEngine: %w", err)
	}
	if _, ok := engine.(wgengine.ResolvingEngine).GetResolver(); !ok {
		panic("internal error: exit node resolver not wired up")
	}

	ns, err := newNetstack(logf, dialer, engine)
	if err != nil {
		return fmt.Errorf("newNetstack: %w", err)
	}
	ns.ProcessLocalIPs = useNetstack
	ns.ProcessSubnets = useNetstack || wrapNetstack

	if useNetstack {
		dialer.UseNetstackForIP = func(ip netip.Addr) bool {
			_, ok := engine.PeerForIP(ip)
			return ok
		}
		dialer.NetstackDialTCP = func(ctx context.Context, dst netip.AddrPort) (net.Conn, error) {
			return ns.DialContextTCP(ctx, dst)
		}
	}

	engine = wgengine.NewWatchdog(engine)
	opts := tsd.ipnServerOpts()

	// store, err := ipnserver.StateStore(filepath.Join(tsd.statePath, "tailscaled.state"), logf)
	store, err := store.New(logf, filepath.Join(tsd.statePath, "tailscaled.state"))
	if err != nil {
		return fmt.Errorf("store.New: %w", err)
	}
	srv, err := ipnserver.New(logf, pol.PublicID.String(), store, engine, dialer, nil, opts)
	if err != nil {
		return fmt.Errorf("ipnserver.New: %w", err)
	}
	ns.SetLocalBackend(srv.LocalBackend())
	if err := ns.Start(); err != nil {
		cb_logger.ErrorMessage("TailscaleDaemon.run(): Failed to start netstack: %v", err)
	}
	
	ln, _, err := safesocket.Listen(tsd.socketPath, safesocket.WindowsLocalPort)
	if err != nil {
		return fmt.Errorf("safesocket.Listen: %v", err)
	}

	tsd.exit.Add(1)
	go func() {
		err = srv.Run(tsd.ctx, ln)
		// Cancelation is not an error: it is the only way to stop ipnserver.
		if err != nil && err != context.Canceled {
			cb_logger.ErrorMessage("TailscaleDaemon.run(): Failed to start IPN server: %v", err)
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

func  (tsd *TailscaleDaemon) createEngine(
	logf logger.Logf, 
	linkMon *monitor.Mon, 
	dialer *tsdial.Dialer,
) (wgengine.Engine, bool, error) {

	var (
		err error
		errs []error

		engine wgengine.Engine
		useNetstack bool
	)

	if tsd.tunname == "" {
		return nil, false, errors.New("no --tun value specified")
	}
	for _, name := range strings.Split(tsd.tunname, ",") {
		logf("wgengine.NewUserspaceEngine(tun %q) ...", name)
		if engine, useNetstack, err = tsd.tryEngine(logf, linkMon, dialer, name); err == nil {
			return engine, useNetstack, nil
		}
		logf("wgengine.NewUserspaceEngine(tun %q) error: %v", name, err)
		errs = append(errs, err)
	}
	return nil, false, multierror.New(errs)
}

func  (tsd *TailscaleDaemon) tryEngine(
	logf logger.Logf, 
	linkMon *monitor.Mon, 
	dialer *tsdial.Dialer, 
	name string,
) (wgengine.Engine, bool, error) {

	var (
		err error

		dev     tun.Device
		devName string

		engine wgengine.Engine
		useNetstack bool
	)

	conf := wgengine.Config{
		ListenPort:  tsd.port,
		LinkMonitor: linkMon,
		Dialer:      dialer,
	}

	useNetstack = name == "userspace-networking"
	netns.SetEnabled(!useNetstack)

	if !useNetstack {
		dev, devName, err = tstun.New(logf, name)
		if err != nil {
			tstun.Diagnose(logf, name, err)
			return nil, false, fmt.Errorf("tstun.New(%q): %w", name, err)
		}
		conf.Tun = dev
		
		if strings.HasPrefix(name, "tap:") {
			conf.IsTAP = true

		} else {
			r, err := router.New(logf, dev, linkMon)
			if err != nil {
				dev.Close()
				return nil, false, fmt.Errorf("router.New: %w", err)
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
		}
	}
	engine, err = wgengine.NewUserspaceEngine(logf, conf)
	if err != nil {
		return nil, useNetstack, err
	}

	go func() {
		tsd.mx.Lock()
		defer tsd.mx.Unlock()

		tsd.devName = devName
		tsd.wgDevice = wgengine.GetWireguardDevice(engine)
	}()
	
	return engine, useNetstack, nil
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

func newNetstack(logf logger.Logf, dialer *tsdial.Dialer, e wgengine.Engine) (*netstack.Impl, error) {
	tunDev, magicConn, dns, ok := e.(wgengine.InternalsGetter).GetInternals()
	if !ok {
		return nil, fmt.Errorf("%T is not a wgengine.InternalsGetter", e)
	}
	return netstack.Create(logf, tunDev, e, magicConn, dialer, dns)
}
