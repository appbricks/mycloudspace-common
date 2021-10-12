// +build darwin

package vpn

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"regexp"

	homedir "github.com/mitchellh/go-homedir"

	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/network"
	"github.com/mevansam/goutils/run"
)

var defaultGatewayPattern = regexp.MustCompile(`^default\s+([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)\s+\S+\s+(\S+[0-9]+)`)

func (w *wireguard) configureNetwork() error {

	var (
		err error

		home,
		line,
		defaultGateway,
		defaultDevice string

		matches [][]string
		
		killall,
		netstat,
		networksetup,
		ifconfig, 
		route run.CLI

		tunIP  net.IP
		tunNet *net.IPNet

		outputBuffer bytes.Buffer
	)

	// List of commands to run to configure 
	// tunnel interface and routes
	//
	// local network's gateway to the internet: 192.168.1.1
	// local tunnel IP for LHS of tunnel: 192.168.111.194
	// peer tunnel IP for RHS of tunnel which is also the tunnel's internet gateway: 192.168.111.1
	// external IP of wireguard peer: 34.204.21.102
	//
	// * configure tunnel network interface
	// 			/sbin/ifconfig utun6 inet 192.168.111.194/32 192.168.111.194 up
	// * configure route to wireguard overlay network via tunnel network interface
	// 			/sbin/route add -inet -net 192.168.111.1 -interface utun6
	// * configure route to peer's public endpoint via network interface connected to the internet
	// 			/sbin/route add inet -net 34.204.21.102 192.168.1.1 255.255.255.255
	// * configure route to send all other traffic through the tunnel by create two routes splitting
	//   the entire IPv4 range of 0.0.0.0/0. i.e. 0.0.0.0/1 and 128.0.0.0/1
	// 			/sbin/route add inet -net 0.0.0.0 192.168.111.1 128.0.0.0
	// 			/sbin/route add inet -net 128.0.0.0 192.168.111.1 128.0.0.0
	//
	// * cleanup
	// 			/sbin/route delete inet -net 34.204.21.102

	home, _ = homedir.Dir()
	null, _ := os.Open(os.DevNull)

	if killall, err = run.NewCLI("/usr/bin/killall", home, &outputBuffer, &outputBuffer); err != nil {
		return err
	}
	if netstat, err = run.NewCLI("/usr/sbin/netstat", home, &outputBuffer, &outputBuffer); err != nil {
		return err
	}
	if networksetup, err = run.NewCLI("/usr/sbin/networksetup", home, &outputBuffer, &outputBuffer); err != nil {
		return err
	}
	if ifconfig, err = run.NewCLI("/sbin/ifconfig", home, null, null); err != nil {
		return err
	}
	if route, err = run.NewCLI("/sbin/route", home, null, null); err != nil {
		return err
	}

	// retrieve current routing table
	if err = netstat.Run([]string{ "-nrf", "inet" }); err != nil {
		return err
	}
	// determine default gateway
	scanner := bufio.NewScanner(bytes.NewReader(outputBuffer.Bytes()))
	for scanner.Scan() {
		line = scanner.Text()
		if matches = defaultGatewayPattern.FindAllStringSubmatch(line, -1); matches != nil && len(matches[0]) == 3 {
			defaultGateway = matches[0][1]
			defaultDevice = matches[0][2]
			break;
		}
	}
	if len(defaultGateway) == 0 || len(defaultDevice) == 0 {
		return fmt.Errorf("unable to determine default gateway and/or device for network client is connected to")
	}
	// retrieve system services
	outputBuffer.Reset()
	if err = networksetup.Run([]string{ "-listallhardwareports" }); err != nil {
		return err
	}
	// determine network service for default device
	matchDevice := "Device: " + defaultDevice
	prevLine := ""
	scanner = bufio.NewScanner(bytes.NewReader(outputBuffer.Bytes()))
	for scanner.Scan() {
		line = scanner.Text()
		if line == matchDevice && len(prevLine) > 0 {
			w.sysDevName  = prevLine[15:]
			break;
		}
		prevLine = line
	}
	if len(w.sysDevName ) == 0 {
		return fmt.Errorf("unable to determine default gateway network client is connected to")
	}
	
	if tunIP, tunNet, err = net.ParseCIDR(w.cfg.tunAddress); err != nil {
		return err
	}
	size, _ := tunNet.Mask.Size()
	if (size == 32) {
		// default to a /24 if address 
		// does not indicate network
		tunNet.Mask = net.CIDRMask(24, 32)
	}

	tunGatewayIP := tunIP.Mask(tunNet.Mask);
	network.IncIP(tunGatewayIP)
	tunGatewayAddress := tunGatewayIP.String()

	// add tunnel IP to local tunnel interface
	if err = ifconfig.Run([]string{ w.ifaceName, "inet", w.cfg.tunAddress, tunIP.String(), "up" }); err != nil {
		return err
	}	
	// create route to tunnel gateway via tunnel interface
	if err = route.Run([]string{ "add", "-inet", "-net", tunGatewayAddress, "-interface", w.ifaceName }); err != nil {
		return err
	}
	// create external routes to peer endpoints
	for _, peerExtAddress := range w.cfg.peerAddresses {
		if err = route.Run([]string{ "add", "-inet", "-net", peerExtAddress, defaultGateway, "255.255.255.255" }); err != nil {
			return err
		}
	}
	// create default route via tunnel gateway
	if err = route.Run([]string{ "add", "-inet", "-net", "0.0.0.0", tunGatewayAddress, "128.0.0.0" }); err != nil {
		return err
	}
	if err = route.Run([]string{ "add", "-inet", "-net", "128.0.0.0", tunGatewayAddress, "128.0.0.0" }); err != nil {
		return err
	}
	// set DNS
	if err = networksetup.Run([]string{ "-setdnsservers", w.sysDevName, w.cfg.tunDNS }); err != nil {
		return err
	}
	// flush DNS cache
	if err = killall.Run([]string{ "-HUP", "mDNSResponder" }); err != nil {
		return err
	}

	return nil
}

func (w *wireguard) cleanupNetwork(resetDefault bool) {

	var (
		err  error
		home string

		route,
		networksetup run.CLI
	)

	home, _ = homedir.Dir()
	null, _ := os.Open(os.DevNull)

	if networksetup, err = run.NewCLI("/usr/sbin/networksetup", home, null, null); err == nil {
		// delete DNS 
		if err = networksetup.Run([]string{ "-setdnsservers", w.sysDevName, "Empty" }); err != nil {
			logger.DebugMessage("ERROR deleting DNS from network service %s: %s", w.sysDevName, err.Error())
		}
	} else {
		logger.DebugMessage("ERROR creating /usr/sbin/networksetup runner: %s", err.Error())
	}

	if route, err = run.NewCLI("/sbin/route", home, null, null); err == nil {
		// delete external routes to peer endpoints
		for _, peerExtAddress := range w.cfg.peerAddresses {
			if err = route.Run([]string{ "delete", "-inet", "-net", peerExtAddress }); err != nil {
				logger.DebugMessage("ERROR deleting route to %s: %s", peerExtAddress, err.Error())
			}
		}

		if resetDefault {
			_ = route.Run([]string{ "delete", "-inet", "-net", "0.0.0.0" })
			_ = route.Run([]string{ "delete", "-inet", "-net", "128.0.0.0" })	
		}

	} else {
		logger.DebugMessage("ERROR creating /sbin/route CLI runner: %s", err.Error())
	}
}
