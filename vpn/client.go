package vpn

import (
	"fmt"

	"github.com/appbricks/cloud-builder/target"
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
	Read() error
	Data() []byte
	Name() string
}

// load vpn config for the space target's admin user
func NewConfigFromTarget(
	tgt *target.Target, 
	configData ConfigData,
) (Config, error) {

	var (
		vpnType string
	)

	if !tgt.Recipe.IsBastion() {
		return nil, fmt.Errorf(fmt.Sprintf("target \"%s\" is not a bastion node", tgt.Key()))
	}
	if output, ok := (*tgt.Output)["cb_vpn_type"]; ok {
		if vpnType, ok = output.Value.(string); !ok {
			return nil, fmt.Errorf(fmt.Sprintf("target's \"cb_vpn_type\" output was not a string: %#v", output))
		}
	}
	switch vpnType {
	case "wireguard":
		return newWireguardConfigFromTarget(configData)
	case "openvpn":
		return newOpenVPNConfigFromTarget(configData)
	default:
		return nil, fmt.Errorf(fmt.Sprintf("target vpn type \"%s\" is not supported", vpnType))
	}
}
