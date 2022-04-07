// +build windows

package vpn

import (
	"context"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mitchellh/go-homedir"

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

		nullOut      *os.File
		outputBuffer bytes.Buffer
	
		executablePath,
		wireguardEXEPath string

		where run.CLI
	)

	home, _ := homedir.Dir()

	if nullOut, err = os.Open(os.DevNull); err != nil {
		logger.ErrorMessage("vpn.init(): Error getting the null file output handle: %s", err.Error())
		panic(err)
	}

	// look for wireguard service exe alongside cb client app
	if executablePath, err = os.Executable(); err != nil {
		logger.ErrorMessage("vpn.init(): Error unable to determine path of CB CLI: %s", err.Error())
		return	
	}
	wireguardEXEPath = filepath.Join(filepath.Dir(executablePath), "wireguard.exe")
	if wireguardEXE, err = run.NewCLI(wireguardEXEPath, home, nullOut, nullOut); err != nil {

		// look for wireguard service exe in system path
		if where, err = run.NewCLI("C:/Windows/System32/where.exe", home, &outputBuffer, &outputBuffer); err != nil {
			logger.ErrorMessage("vpn.init(): Error creating CLI for \"where\" command: %s", err.Error())
			return
		}
		if err = where.Run([]string{ "wireguard.exe" }); err != nil {		
			logger.ErrorMessage("vpn.init(): Error running \"where wireguard.exe\" command: %s", err.Error())
			return
		}
		results := utils.ExtractMatches(outputBuffer.Bytes(), map[string]*regexp.Regexp{
			"wireguardPath": regexp.MustCompile(`^.*\\wireguard.exe$`),
		})
		if len(results["wireguardPath"]) > 0 && len(results["wireguardPath"][0]) == 1 {
			wireguardEXEPath = results["wireguardPath"][0][0]
			if wireguardEXE, err = run.NewCLI(wireguardEXEPath, home, nullOut, nullOut); err != nil {
				logger.ErrorMessage("vpn.init(): Error creating CLI for \"%s\" command: %s", wireguardEXEPath, err.Error())
				return
			}

		} else {
			logger.ErrorMessage("vpn.init(): Unable to locate wireguard service executable.")	
			return
		}
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
