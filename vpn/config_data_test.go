package vpn_test

import (
	"github.com/appbricks/mycloudspace-common/vpn"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("VPN Configuration", func() {

	var (
		// err error

		testService *testService
	)

	BeforeEach(func() {
		testService = startTestService()
	})

	AfterEach(func() {
		testService.stop()
	})

	It("loads a static vpn configuration", func() {
		testService.httpTestSvrExpectedURI = "/static/~bastion-admin/mycs-test.conf"

		configData, err := vpn.NewVPNConfigData(testService)
		Expect(err).ToNot(HaveOccurred())
		Expect(configData).ToNot(BeNil())
		Expect(configData.Name()).To(Equal("mycs-test"))
		Expect(string(configData.Data())).To(Equal(wireguardConfig))
	})

	It("loads a dynamic vpn configuration", func() {

		testService.parseConnectResponse(connectResponse)

		configData, err := vpn.NewVPNConfigData(testService)
		Expect(err).ToNot(HaveOccurred())
		Expect(configData).ToNot(BeNil())
		Expect(configData.Name()).To(Equal("dynamic config"))
		Expect(string(configData.Data())).To(Equal(dynamicWireguardConfig))
	})
})

const connectResponse = `{
  "name": "dynamic config",
  "vpnType": "wireguard",
  "config": {
    "client_addr": "192.168.111.1",
    "dns": "10.12.16.253",
    "peer_endpoint": "test-us-east-1.aws.appbricks.io:3399",
    "peer_public_key": "public key",
    "allowed_subnets": [
      "0.0.0.0/0"
    ],
    "keep_alive_ping": 25
  }
}`

const dynamicWireguardConfig = `[Interface]
PrivateKey = private key
Address = 192.168.111.1/32
DNS = 10.12.16.253

[Peer]
PublicKey = public key
Endpoint = test-us-east-1.aws.appbricks.io:3399
PersistentKeepalive = 25
AllowedIPs = 0.0.0.0/0
`