// +build windows

package vpn

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/run"
	"github.com/mevansam/goutils/utils"
	"github.com/appbricks/mycloudspace-common/monitors"
)

type wireguard struct {	
	cfg *wireguardConfig

	tunnelName,
	configFilePath string

	wgctrlClient *WGCtrlClient

	// bytes sent and received through the tunnel
	sent, recd *monitors.Counter

	metricsTimer *utils.ExecTimer
	metricsError error
}

var (
	wireguardEXE run.CLI
)

func init() {

	var (
		err error

		nullOut *os.File
	
		executablePath string
	)

	if nullOut, err = os.Open(os.DevNull); err != nil {
		logger.ErrorMessage("vpn.init(): Error getting the null file output handle: %s", err.Error())
		panic(err)
	}

	// look for wireguard service exe alongside cb client app
	if executablePath, err = os.Executable(); err != nil {
		logger.ErrorMessage("vpn.init(): Error unable to determine path of CB CLI: %s", err.Error())
		return	
	}
	run.AddCliSearchPaths("wireguard.exe", filepath.Dir(executablePath))
	if drives, err := run.GetLogicalDrives(); err == nil {
		for _, d := range drives {
			run.AddCliSearchPaths("wireguard.exe", 
				filepath.Join(d, "Program Files", "WireGuard"),
				filepath.Join(d, "WireGuard"),
			)
		}	
	}
	if wireguardEXE, executablePath, err = run.CreateCLI("wireguard.exe", nullOut, nullOut); err != nil {
		logger.ErrorMessage("vpn.init(): Error unable to determine path of wireguard EXE: %s", err.Error())
		return
	}
}

func newWireguardClient(cfg *wireguardConfig, monitorService *monitors.MonitorService) (*wireguard, error) {

	if wireguardEXE == nil {
		return nil, fmt.Errorf("wireguard exe not found")
	}	
	w := &wireguard{
		cfg:            cfg,
		tunnelName:     cfg.configFileName[:strings.LastIndex(cfg.configFileName, ".")],
		configFilePath: filepath.Join(os.TempDir(), cfg.configFileName),
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
	)

	if err = ioutil.WriteFile(w.configFilePath, w.cfg.configData, 0644); err != nil {
		return err
	}
	logger.DebugMessage("wireguard.Connect(): Wireguard configuration written to temp file: %s", w.configFilePath)

	if err = wireguardEXE.Run([]string{ "/installtunnelservice", w.configFilePath }); err != nil {		
		logger.ErrorMessage(
			"vpn.init(): Error running \"wireguard.exe /installtunnelservice %s\" command: %s", 
			w.configFilePath, err.Error(),
		)
		return err
	}
	logger.DebugMessage("wireguard.Connect(): Successfully installed wireguard tunnel service '%s'", w.tunnelName)

	defer func() {
		// if an err is being returned then
		// ensure tunnel and device is closed
		if err != nil {
			w.Disconnect()
		}
	}()

	if w.wgctrlClient, err = NewWireguardCtrlClient(w.tunnelName); err != nil {
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

	if err := wireguardEXE.Run([]string{ "/uninstalltunnelservice", w.tunnelName }); err != nil {		
		logger.ErrorMessage(
			"vpn.init(): Error running \"wireguard.exe /uninstalltunnelservice %s\" command: %s", 
			w.tunnelName, err.Error(),
		)
		return err
	}
	logger.DebugMessage("wireguard.Connect(): Successfully uninstalled wireguard tunnel service '%s'", w.tunnelName)

	// remove wireguard config file
	_ = os.Remove(w.configFilePath)

	return nil
}
