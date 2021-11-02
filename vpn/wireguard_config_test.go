package vpn_test

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"

	"github.com/appbricks/cloud-builder/target"
	"github.com/appbricks/mycloudspace-common/vpn"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	cookbook_mocks "github.com/appbricks/cloud-builder/test/mocks"
	utils_mocks "github.com/mevansam/goutils/test/mocks"
)

var _ = Describe("Wireguard Config", func() {

	var (
		err error

		testTarget *testTarget
		config     vpn.Config

		downloadPath string
	)

	Context("load", func() {

		BeforeEach(func() {
			// test http server to mock bastion HTTPS 
			// backend for vpn config retrieval
			testTarget = startTestTarget()

			downloadPath, err = os.MkdirTemp("", "vpn");
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			testTarget.stop()
			os.RemoveAll(downloadPath)
		})

		It("load wireguard vpn config to connect to a target", func() {
			testTarget.httpTestSvrExpectedURI = "/static/~bastion-admin/mycs-test.conf"

			// ensure target remotes status is loaded
			err = testTarget.target.LoadRemoteRefs()
			Expect(err).NotTo(HaveOccurred())

			config, err = vpn.NewConfigFromTarget(vpn.NewStaticConfigData(testTarget.target, "bastion-admin", ""))
			Expect(err).NotTo(HaveOccurred())
			Expect(config).ToNot(BeNil())
			Expect(reflect.TypeOf(config).String()).To(Equal("*vpn.wireguardConfig"))		
			Expect(config.Config()).To(Equal(wireguardConfig))
			Expect(testTarget.httpTestSvrErr).NotTo(HaveOccurred())		
			
			desc, err := config.Save(downloadPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(desc[0:400]).To(MatchRegexp(fmt.Sprintf(wireguardConfigSave, downloadPath)))
		})
	})
})

type testTarget struct {
	httpTestSvr *httptest.Server

	httpTestSvrExpectedURI string
	httpTestSvrErr error

	target *target.Target
}

func startTestTarget() *testTarget {

	var (
		err error

		cer tls.Certificate
	)

	t := &testTarget{}

	// test http server to mock bastion HTTPS 
	// backend for vpn config retrieval
	cer, err = tls.X509KeyPair([]byte(testTLSServerCert), []byte(testTLSServerKey))
	Expect(err).NotTo(HaveOccurred())
	tlsCfg := &tls.Config{Certificates: []tls.Certificate{cer}}

	t.httpTestSvr = httptest.NewUnstartedServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			user, passwd, ok := r.BasicAuth()
			Expect(ok).To(BeTrue())
			Expect(user).To(Equal("bastion-admin"))
			Expect(passwd).To(Equal(""))
			if r.RequestURI != t.httpTestSvrExpectedURI {
				http.NotFound(w, r)
				return
			}
			_, t.httpTestSvrErr = w.Write([]byte(wireguardConfig))
		},
	))
	t.httpTestSvr.TLS = tlsCfg
	t.httpTestSvr.StartTLS()

	// mock target describing the space for which 
	// vpn configuration is to be retrieved
	tgts := target.NewTargetSet(&fakeTargetContext{})
	err = tgts.UnmarshalJSON(
		[]byte(
			fmt.Sprintf(targets, strings.Split(t.httpTestSvr.URL, ":")[2]),
		),
	)
	t.target = tgts.GetTarget("fakeRecipe/fakeIAAS/")
	Expect(err).NotTo(HaveOccurred())	
	return t
}

func (t *testTarget) URL() string {
	return t.httpTestSvr.URL
}

func (t *testTarget) stop() {
	t.httpTestSvr.Close()
}

type fakeTargetContext struct {
	outputBuffer, errorBuffer strings.Builder
}

func (ctx *fakeTargetContext) NewTarget(
	recipeName, recipeIaas string,
) (*target.Target, error) {

	var (
		cli *utils_mocks.FakeCLI		
		tgt *target.Target
	)

	cli = utils_mocks.NewFakeCLI(&ctx.outputBuffer, &ctx.errorBuffer)			
	tgt = cookbook_mocks.NewMockTarget(cli, "1.1.1.1", 9999, "")
	tgt.Recipe.(*cookbook_mocks.FakeRecipe).SetBastion()

	return tgt, nil
}

const testTLSServerCert = `-----BEGIN CERTIFICATE-----
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
-----END CERTIFICATE-----`
const testTLSServerKey = `-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----`
const wireguardConfig = `[Interface]
PrivateKey = gCgKmNwxEtBo0Y0oZVgkOABuRBfafoWMk8sb9yiHCGQ=
Address = 192.168.111.194/32
DNS = 10.0.16.253

[Peer]
PublicKey = AnTKCPYQCkNACBUsB2otfk+V/D3ZiBpNaQJHsSw0hEo=
AllowedIPs = 0.0.0.0/0
Endpoint = 34.204.21.102:3399
PersistentKeepalive = 25
`
const wireguardConfigSave = `The VPN configuration has been downloaded to the file shown below.
You need import it to the wireguard vpn client via the option "Import
Tunnels from file...".

%s/mycs-test.conf

Scan the following QR code with the mobile client to configure the
VPN on you mobile device.
.*`

const targets = `[
	{
		"recipeName": "fakeRecipe",
		"recipeIaas": "fakeIAAS",
		"dependentTargets": [			
		],
		"recipe": {
			"variables": []
		},
		"provider": {
			"access_key": "mycs-test-aws-key",
			"secret_key": "mycs-test-aws-secret",
			"region": "us-east-1",
			"token": ""
		},
		"backend": {
			"bucket": "mycs-test-bucket",
			"key": "sandbox"
		},
		"output": {
			"cb_managed_instances": {
				"Sensitive": false,
				"Type": [
					"tuple",
					[
						[
							"object",
							{
								"description": "string",
								"fqdn": "string",
								"id": "string",
								"name": "string",
								"order": "number",
								"private_ip": "string",
								"public_ip": "string",
								"root_user": "string",
								"root_passwd": "string",
								"non_root_user": "string",
								"non_root_passwd": "string",
								"ssh_key": "string",
								"ssh_port": "string",
								"ssh_user": "string",
								"api_port": "string"
							}
						]
					]
				],
				"Value": [
					{
						"description": "",
						"fqdn": "",
						"id": "bastion-instance-id",
						"name": "bastion",
						"order": 0,
						"private_ip": "127.0.0.1",
						"public_ip": "127.0.0.1",
						"root_user": "bastion-admin",
						"root_passwd": "root_p@ssw0rd",
						"non_root_user": "bastion-user",
						"non_root_passwd": "user_p@ssw0rd",
						"ssh_key": "",
						"ssh_port": "22",
						"ssh_user": "bastion-admin",
						"api_port": "%s"
					}
				]
			},
			"cb_root_ca_cert": {
				"Sensitive": false,
				"Type": "string",
				"Value": "-----BEGIN CERTIFICATE-----\n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\nwJ7PCAkh\n-----END CERTIFICATE-----"
			},
			"cb_vpc_id": {
				"Sensitive": false,
				"Type": "string",
				"Value": "vpc-id"
			},
			"cb_vpc_name": {
				"Sensitive": false,
				"Type": "string",
				"Value": "mycs-test"
			},
			"cb_vpn_type": {
				"Sensitive": false,
				"Type": "string",
				"Value": "wireguard"
			},
			"cb_vpn_type": {
				"Sensitive": false,
				"Type": "string",
				"Value": "wireguard"
			}
		},
		"cookbook_timestamp": "1614567035"
	}
]`
