package vpn

import (
	"encoding/json"
	"fmt"

	"github.com/appbricks/cloud-builder/userspace"
	"github.com/appbricks/mycloudspace-common/monitors"
)

type ServiceConfig struct {
	PrivateKey string
	PublicKey  string
	
	IsAdminUser  bool

	Name    string `json:"name,omitempty"`
	VPNType string `json:"vpnType,omitempty"`
	
	RawConfig json.RawMessage `json:"config,omitempty"`
}

type Service interface {
	Connect() (*ServiceConfig, error)
	Disconnect() error

	GetSpaceNode() userspace.SpaceNode
}

type ConfigData interface {	
	Name() string
	VPNType() string
	Data() []byte
	Delete() error
}

type Config interface {
	NewClient(monitorService *monitors.MonitorService) (Client, error)
	Config() string

	Save(path string) (string, error)
}

type Client interface {
	Connect() error
	Disconnect() error
	
	BytesTransmitted() (int64, int64, error)
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
