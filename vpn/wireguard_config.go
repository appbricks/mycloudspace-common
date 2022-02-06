package vpn

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/appbricks/mycloudspace-common/monitors"
	qrcode "github.com/skip2/go-qrcode"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type wireguardConfig struct {	
	configFileName string
	configData     []byte

	tunAddress,
	tunDNS string

	peerAddresses  []string
	isDefaultRoute bool
	
	config wgtypes.Config
}

var configSectionPattern = regexp.MustCompile(`\[(.*)\]`)
var configAttribPattern = regexp.MustCompile(`(^[a-zA-Z0-9\-\_]*)\s*=\s*(.*)`)

func newWireguardConfigFromTarget(configData ConfigData) (*wireguardConfig, error) {

	var (
		err error	

		line    string
		matches [][]string

		key, value string
		peerConfig wgtypes.PeerConfig
	)
	c := &wireguardConfig{}

	isInterface := false
	isPeer := false

	peerConfig = wgtypes.PeerConfig{}

	c.configData = configData.Data()
	c.configFileName = fmt.Sprintf("%s.conf", configData.Name())
	
	scanner := bufio.NewScanner(bytes.NewReader(c.configData))
	for scanner.Scan() {
		line = scanner.Text()
		
		if matches = configSectionPattern.FindAllStringSubmatch(line, -1); matches != nil && len(matches[0]) > 0 {
			if isPeer {
				c.config.Peers = append(c.config.Peers, peerConfig)
				peerConfig = wgtypes.PeerConfig{}
			}
			switch matches[0][1] {
				case "Interface":
					isInterface = true
					isPeer = false
				case "Peer":
					isInterface = false
					isPeer = true
				default:
					isInterface = false
					isPeer = false
			}
			continue

		} else if matches = configAttribPattern.FindAllStringSubmatch(line, -1); matches != nil && len(matches[0]) > 0 {
			key = matches[0][1]
			value = matches[0][2]
		}

		if len(key) > 0 {
			if isInterface {
				switch key {
					case "PrivateKey":
						var wgKey wgtypes.Key
						if wgKey, err = wgtypes.ParseKey(value); err != nil {
							return nil, err
						}
						c.config.PrivateKey = &wgKey
					case "Address":
						c.tunAddress = value
					case "DNS":
						c.tunDNS = value
					default:
						return nil, 
							fmt.Errorf(
								"wireguard client config key '%s' within the 'Interface' topic is not supported", 
								key,
							)
				}
			} else if isPeer {
				switch key {
					case "PublicKey":
						if peerConfig.PublicKey, err = wgtypes.ParseKey(value); err != nil {
							return nil, err
						}
					case "AllowedIPs":
						var ipNet *net.IPNet
						ips := strings.Split(value, ",")
						for _, ip := range ips {
							if !c.isDefaultRoute && ip == "0.0.0.0/0" {
								// set this tunnel as handling all network traffic that
								// does not have an explicit routing rule. i.e. all 
								// network traffic is tunneled via this wireguard tunnel.
								// wireguard internally routes traffic to the correct
								// peer
								c.isDefaultRoute = true
							}
							if _, ipNet, err = net.ParseCIDR(ip); err != nil {
								return nil, err
							}
							peerConfig.AllowedIPs = append(peerConfig.AllowedIPs, *ipNet)
						}
					case "Endpoint":
						if peerConfig.Endpoint, err = net.ResolveUDPAddr("udp", value); err != nil {
							return nil, err
						}
						c.peerAddresses = append(c.peerAddresses, string(peerConfig.Endpoint.IP.String()))
					case "PersistentKeepalive":
						var v int64
						if v, err = strconv.ParseInt(value, 10, 64); err != nil {
							return nil, err
						}
						keepAlive := time.Duration(v)
						peerConfig.PersistentKeepaliveInterval = &keepAlive
					default:
						return nil, 
							fmt.Errorf(
								"wireguard client config key '%s' within the 'Peer' topic is not supported", 
								key,
							)
				}
			}
			key = ""
			value = ""
		}
	}
	if isPeer {
		c.config.Peers = append(c.config.Peers, peerConfig)
		peerConfig = wgtypes.PeerConfig{}
	}

	return c, nil
}

func (c *wireguardConfig) NewClient(monitorService *monitors.MonitorService) (Client, error) {
	return newWireguardClient(c, monitorService)
}

func (c *wireguardConfig) Config() string {
	return string(c.configData)
}

func (c *wireguardConfig) Save(path string) (string, error) {
	
	var (
		err error

		qrCode *qrcode.QRCode
	)
	downloadFilePath := filepath.Join(path, c.configFileName)
	
	if err = ioutil.WriteFile(downloadFilePath, c.configData, 0644); err != nil {
		return "", err
	}
	if qrCode, err = qrcode.New(string(c.configData), qrcode.Low); err != nil {
		return "", err
	}
	qrCode.DisableBorder = true

	const desc = `The VPN configuration has been downloaded to the file shown below.
You need import it to the wireguard vpn client via the option "Import
Tunnels from file...".

%s

Scan the following QR code with the mobile client to configure the
VPN on you mobile device.

%s`
	return fmt.Sprintf(
		desc, 
		downloadFilePath,
		qrCode.ToSmallString(false),
	), nil
}
