package vpn

import (
	"encoding/json"
	"fmt"

	"github.com/appbricks/cloud-builder/target"
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

	Save(path, prefix string) (string, error)
}

type Client interface {
	Connect() error
	Disconnect() error
	
	BytesTransmitted() (int64, int64, error)
}

// retrieve vpn configuration from the target
func NewVPNConfigData(service Service) (ConfigData, error) {

	var (
		err error
		ok  bool

		cfg *ServiceConfig
		tgt *target.Target

		userName, password string
	)

	if cfg, err = service.Connect(); err != nil {
		return nil, err
	}

	if cfg.RawConfig != nil {
		switch cfg.VPNType {
		case "wireguard":
			wgConfigData := &wireguardConfigData{
				service:    service,
				name:       cfg.Name,
				privateKey: cfg.PrivateKey,
			}
			if err = json.Unmarshal(cfg.RawConfig, wgConfigData); err != nil {
				return nil, err
			}
			return wgConfigData, nil
	
		default:
			return nil, fmt.Errorf("unknown VPN type \"%s\"", cfg.VPNType)
		}

	} else {
		// if no VPN config provided then attempt
		// to download a static configuration

		if tgt, ok = service.GetSpaceNode().(*target.Target); !ok {
			return nil, fmt.Errorf("cannot connect to a space node that is not an owned target")
		}
		instance := tgt.ManagedInstance("bastion")
		if instance == nil {
			return nil, fmt.Errorf("space target \"%s\" does not have a deployed bastion instance.", tgt.Key())
		}

		if cfg.IsAdminUser {
			userName = instance.RootUser()
			password = instance.RootPassword()
		} else {
			userName = instance.NonRootUser()
			password = instance.NonRootPassword()
		}
		return newStaticConfigData(tgt, userName, password)
	}
}

// initialize the vpn client configuration
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
