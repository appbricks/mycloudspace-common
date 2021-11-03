package vpn

import (
	"fmt"
)

type Config interface {
	NewClient() (Client, error)
	Config() string

	Save(path string) (string, error)
}

type Client interface {
	Connect() error
	Disconnect() error
	
	BytesTransmitted() (int64, int64, error)
}

type ConfigData interface {	
	Name() string
	VPNType() string
	Data() []byte
	Delete() error
}

// load vpn config for the space target's admin user
func NewConfigFromTarget(configData ConfigData) (Config, error) {
	vpnType := configData.VPNType()
	switch vpnType {
	case "wireguard":
		return newWireguardConfigFromTarget(configData)
	case "openvpn":
		return newOpenVPNConfigFromTarget(configData)
	default:
		return nil, fmt.Errorf(fmt.Sprintf("target vpn type \"%s\" is not supported", vpnType))
	}
}
