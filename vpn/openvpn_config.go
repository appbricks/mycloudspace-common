package vpn

import (
	"fmt"

	"github.com/appbricks/cloud-builder/target"
)

type openvpnConfig struct {	
}

func newOpenVPNConfigFromTarget(tgt *target.Target, user, passwd string) (*openvpnConfig, error) {
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
