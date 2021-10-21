package vpn

import (
	"fmt"
	"net"
	"strings"

	"github.com/mevansam/goutils/logger"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type WGCtrlService struct {	
	ifaceName string

	uapi net.Listener	
	err  chan error

	device       *device.Device
	deviceLogger *device.Logger
}

func NewWireguardCtrlService(
	ifaceName string, 
	device *device.Device, 
	deviceLogger *device.Logger,
) *WGCtrlService {

	return &WGCtrlService{
		ifaceName:    ifaceName,
		err:          make(chan error),
		device:       device,
		deviceLogger: deviceLogger,
	}
}

func (wgcs *WGCtrlService) Start() error {
	return wgcs.startUAPI()
}

func (wgcs *WGCtrlService) Stop() error {

	var (
		err error
	)

	if wgcs.uapi != nil {
		if err = wgcs.uapi.Close(); err != nil {
			logger.DebugMessage("Error closing UAPI socket: %s", err.Error())
		}
		err = <-wgcs.err
		return err
	}
	return nil
}

type WGCtrlClient struct {
	ifaceName string
	wgClient  *wgctrl.Client	
}

func NewWireguardCtrlClient(ifaceName string) (*WGCtrlClient, error) {

	var (
		err error
	)

	wgcc := &WGCtrlClient{
		ifaceName: ifaceName,
	}
	if wgcc.wgClient, err = wgctrl.New(); err != nil {
		return nil, err
	}
	return wgcc, nil
}

func (wgcc *WGCtrlClient) Configure(cfg wgtypes.Config) error {
	return wgcc.wgClient.ConfigureDevice(wgcc.ifaceName, cfg)
}

func (wgcc *WGCtrlClient) StatusText() (string, error) {

	var (
		err error

		status strings.Builder
		device *wgtypes.Device
	)
	
	if device, err = wgcc.wgClient.Device(wgcc.ifaceName); err != nil {
		return "", err
	}

	const deviceStatus = `interface: %s (%s)
  public key: %s
  private key: (hidden)
`
	status.WriteString(
		fmt.Sprintf(
			deviceStatus,
			device.Name,
			device.Type.String(),
			device.PublicKey.String(),
		),
	)

	const peerStatus = `
peer: %s
  endpoint: %s
  allowed ips: %s
  latest handshake: %s
  transfer: %d B received, %d B sent
`
	for _, peer := range device.Peers {
		allowedIPs := make([]string, 0, len(peer.AllowedIPs))
		for _, ip := range peer.AllowedIPs {
			allowedIPs = append(allowedIPs, ip.String())
		}
		status.WriteString(
			fmt.Sprintf(
				peerStatus,
				peer.PublicKey.String(),
				peer.Endpoint.String(),
				strings.Join(allowedIPs, ", "),
				peer.LastHandshakeTime.String(),
				peer.ReceiveBytes,
				peer.TransmitBytes,
			),
		)
	}
	return status.String(), nil
}

func (wgcc *WGCtrlClient) BytesTransmitted() (int64, int64, error) {

	var (
		err error
		
		device     *wgtypes.Device
		sent, recd int64
	)
	
	if device, err = wgcc.wgClient.Device(wgcc.ifaceName); err != nil {
		return -1, -1, err
	}

	recd = 0
	sent = 0
	for _, peer := range device.Peers {
		recd += peer.ReceiveBytes
		sent += peer.TransmitBytes
	}
	return recd, sent, nil
}
