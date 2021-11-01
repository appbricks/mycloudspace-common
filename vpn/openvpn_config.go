package vpn

import (
	"fmt"
)

type openvpnConfig struct {	
}

func newOpenVPNConfigFromTarget(configData ConfigData) (*openvpnConfig, error) {
	return &openvpnConfig{}, fmt.Errorf("openvpn client connect is not supported")
}

func (c *openvpnConfig) NewClient() (Client, error) {
	return newOpenVPNClient(c)
}

func (c *openvpnConfig) Config() string {
	return ""
}

func (c *openvpnConfig) Save(path string) (string, error) {
	return "", nil
}
