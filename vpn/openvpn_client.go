package vpn

type openvpn struct {	
}

func newOpenVPNClient(cfg *openvpnConfig) (*openvpn, error) {
	return &openvpn{}, nil
}

func (o *openvpn) Connect() error {
	return nil
}

func (o *openvpn) Disconnect() error {
	return nil
}

func (o *openvpn) StatusText() (string, error) {
	return "", nil
}

func (w *openvpn) BytesTransmitted() (int64, int64, error) {
	return 0, 0, nil
}