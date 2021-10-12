// +build linux

package vpn

import (
	"golang.zx2c4.com/wireguard/device"
)

func (w *wireguard) startUAPI(deviceLogger *device.Logger) error {
	return nil
}

func (w *wireguard) configureNetwork() error {
	return nil
}

func (w *wireguard) cleanupNetwork(resetDefault bool) {
}
