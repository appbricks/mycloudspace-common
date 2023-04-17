package vpn

import (
	"fmt"

	"github.com/appbricks/mycloudspace-common/monitors"
)

type openvpnConfig struct {	
}

func newOpenVPNConfigFromTarget(configData ConfigData) (*openvpnConfig, error) {
	return &openvpnConfig{}, fmt.Errorf("openvpn client connect is not supported")
}

func (c *openvpnConfig) NewClient(monitorService *monitors.MonitorService) (Client, error) {
	return newOpenVPNClient(c)
}

func (c *openvpnConfig) Config() string {
	return ""
}

func (c *openvpnConfig) Save(path, prefix string) (string, error) {
	return "", nil
}
