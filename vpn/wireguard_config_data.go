package vpn

import (
	"bytes"
	"fmt"
	"strings"
)

type wireguardConfigData struct {
	service Service

	name string

	privateKey string
	Address    string `json:"client_addr,omitempty"`
	DNS        string `json:"dns,omitempty"`

	PeerEndpoint   string   `json:"peer_endpoint,omitempty"`
	PeerPublicKey  string   `json:"peer_public_key,omitempty"`
	AllowedSubnets []string `json:"allowed_subnets,omitempty"`
	KeepAlivePing  int      `json:"keep_alive_ping,omitempty"`
}

func (c *wireguardConfigData) Name() string {	
	return c.name
}

func (c *wireguardConfigData) VPNType() string {	
	return "wireguard"
}

func (c *wireguardConfigData) Data() []byte {	

	configText := new(bytes.Buffer)

	const interfaceSectionF = `[Interface]
PrivateKey = %s
Address = %s/32
`

	fmt.Fprintf(
		configText,
		interfaceSectionF,
		c.privateKey,
		c.Address,
	)

	if len(c.DNS) > 0 {
		fmt.Fprintf(
			configText, "DNS = %s\n", 
			c.DNS,
		)
	}

	const peerSectionF = `
[Peer]
PublicKey = %s
Endpoint = %s
PersistentKeepalive = %d
`

	fmt.Fprintf(
		configText,
		peerSectionF,
		c.PeerPublicKey,
		c.PeerEndpoint,
		c.KeepAlivePing,
	)

	if len(c.AllowedSubnets) > 0 {
		fmt.Fprintf(
			configText,
			"AllowedIPs = %s\n", 
			strings.Join(c.AllowedSubnets, ","),
		)
	}

	return configText.Bytes()
}

func (c *wireguardConfigData) Delete() error {
	return c.service.Disconnect()
}
