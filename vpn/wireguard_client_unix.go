//go:build linux || darwin
// +build linux darwin

package vpn

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"

	"github.com/appbricks/mycloudspace-common/monitors"
	log "github.com/sirupsen/logrus"

	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/network"
	"github.com/mevansam/goutils/utils"
)

type wireguard struct {	

	cfg *wireguardConfig
	nc  network.NetworkContext

	ifaceName string

	tunnel tun.Device
	device *device.Device

	wgctrlService *WGCtrlService
	wgctrlClient  *WGCtrlClient

	close        chan bool
	disconnected chan bool

	// bytes sent and received through the tunnel
	sent, recd *monitors.Counter

	metricsTimer *utils.ExecTimer
	metricsError error
}

func newWireguardClient(cfg *wireguardConfig, monitorService *monitors.MonitorService) (*wireguard, error) {

	var (
		err error

		nc network.NetworkContext
	)

	if nc, err = network.NewNetworkContext(); err != nil {
		return nil, err
	}

	w := &wireguard{
		cfg: cfg,
		nc:  nc,

		close:        make(chan bool),
		disconnected: make(chan bool),
	}

	w.sent = monitors.NewCounter("sent", true, true)
	w.recd = monitors.NewCounter("recd", true, true)

	// create monitors
	if monitorService != nil {
		monitor := monitorService.NewMonitor("space-vpn")
		monitor.AddCounter(w.sent)
		monitor.AddCounter(w.recd)	
	}

	return w, nil
}

func (w *wireguard) Connect() error {

	var (
		err error

		tunIfaceName string
		dnsManager   network.DNSManager
		routeManager network.RouteManager
		tunRoute     network.RoutableInterface
	)

	logLevel := func() int {
		switch log.GetLevel() {
		case log.TraceLevel, log.DebugLevel:
			return device.LogLevelVerbose
		case log.InfoLevel, log.WarnLevel, log.ErrorLevel:
			return device.LogLevelError
		}
		return device.LogLevelError
	}()

	// determine tunnnel device name
	if runtime.GOOS == "darwin" {
		w.ifaceName = "utun"
	} else {
		w.ifaceName = "wg"
	}
	if w.ifaceName, err = network.GetNextAvailabeInterface(w.ifaceName); err != nil {
		return err
	}
	// open TUN device on utun#
	if w.tunnel, err = tun.CreateTUN(w.ifaceName, device.DefaultMTU); err != nil {
		logger.ErrorMessage("wireguard.Connect(): Failed to create TUN device: %s", err.Error())
		return err
	}
	if tunIfaceName, err = w.tunnel.Name(); err == nil {
		w.ifaceName = tunIfaceName
	}

	deviceLogger := device.NewLogger(
		logLevel,
		fmt.Sprintf("(%s) ", w.ifaceName),
	)
	deviceLogger.Verbosef("Starting mycs wireguard tunnel")
	
	w.device = device.NewDevice(w.tunnel, conn.NewDefaultBind(), deviceLogger)
	defer func() {
		// if an err is being returned then
		// ensure tunnel and device is closed
		if err != nil {
			logger.ErrorMessage("wireguard.Connect(): Exited with error: %s", err.Error())
			w.tunnel.Close()
			w.device.Close()
		}
	}()

	if err = w.device.Up(); err != nil {
		return err
	}
	deviceLogger.Verbosef("Device started")

	w.wgctrlService = NewWireguardCtrlService(w.ifaceName, w.device)
	if err = w.wgctrlService.Start(); err != nil {
		return err
	}
	if w.wgctrlClient, err = NewWireguardCtrlClient(w.ifaceName); err != nil {
		return err
	}
	
	// handle termination of services
	go func() {
		var (
			err error
		)

		// stop recieving interrupt
		// signals on channel
		defer func() {
			w.device.Close()
			w.disconnected <- true
		}()

		select {
			case <-w.close:
			case <-w.device.Wait():
		}		
		deviceLogger.Verbosef("Shutting down wireguard tunnel")

		if err = w.wgctrlService.Stop(); err != nil {
			logger.DebugMessage("wireguard.Connect(): Error closing UAPI socket: %s", err.Error())
		}
		if err = w.tunnel.Close(); err != nil {
			logger.DebugMessage("wireguard.Connect(): Error closing TUN device: %s", err.Error())
		}
		// cleanup dns and routing
		w.nc.Clear()

		logger.DebugMessage("wireguard.Connect(): Wireguard client has been disconnected.")
	}()

	// configure the wireguard tunnel
	if err = w.wgctrlClient.Configure(w.cfg.config); err != nil {
		return err
	}
	// disable ipv6
	if err = w.nc.DisableIPv6(); err != nil {
		return err
	}
	// configure routing
	if routeManager, err = w.nc.NewRouteManager(); err != nil {
		return err
	}
	if err = routeManager.AddExternalRouteToIPs(w.cfg.peerAddresses); err != nil {
		return err
	}
	if tunRoute, err = routeManager.NewRoutableInterface(w.ifaceName, w.cfg.tunAddress); err != nil {
		return err
	}
	if w.cfg.isDefaultRoute {
		if err = tunRoute.MakeDefaultRoute(); err != nil {
			return err
		}	
	}
	// configure dns
	if dnsManager, err = w.nc.NewDNSManager(); err != nil {
		return err
	}
	if err = dnsManager.AddDNSServers([]string{ w.cfg.tunDNS }); err != nil {
		return err
	}
	if err = dnsManager.AddSearchDomains([]string{}); err != nil {
		return err
	}

	// start background thread to record tunnel metrics
	w.metricsTimer = utils.NewExecTimer(context.Background(), w.recordNetworkMetrics, false)
	if err = w.metricsTimer.Start(500); err != nil {
		logger.ErrorMessage(
			"wireguard.Connect(): Unable to start metrics collection job: %s", 
			err.Error(),
		)
	}

	return nil
}

func (w *wireguard) Disconnect() error {
	if w.metricsTimer != nil {
		_ = w.metricsTimer.Stop()
	}

	w.close<-true
	select {
		case <-w.disconnected:		
		case <-time.After(time.Millisecond * 500):
			logger.WarnMessage(
				"wireguard.Disconnect(): Timed out waiting for VPN disconnect signal. Most likely connection was not established.",
			)
			// cleanup dns and routing
			w.nc.Clear()
	}
	return nil
}
