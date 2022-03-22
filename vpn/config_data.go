package vpn

import (
	"encoding/json"
	"fmt"

	"github.com/appbricks/cloud-builder/target"
)

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
