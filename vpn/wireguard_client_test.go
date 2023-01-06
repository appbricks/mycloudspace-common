package vpn_test

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"reflect"
	"regexp"
	"time"

	homedir "github.com/mitchellh/go-homedir"

	"golang.zx2c4.com/wireguard/wgctrl"

	"github.com/appbricks/mycloudspace-common/vpn"
	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/network"
	"github.com/mevansam/goutils/run"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Wireguard Client", func() {

	var (
		testService *testService
		config     vpn.Config
		client     vpn.Client
	)

	Context("create", func() {

		BeforeEach(func() {
			// test http server to mock bastion HTTPS 
			// backend for vpn config retrieval
			testService = startTestService()
		})

		AfterEach(func() {
			testService.stop()
		})

		It("create wireguard vpn client to connect to a target", func() {
			isAdmin, err := run.IsAdmin()
			Expect(err).NotTo(HaveOccurred())
			if !isAdmin {
				Fail("This test needs to be run with root privileges. i.e. sudo -E go test -v ./...")
			}

			var (
				tunIfaceName string

				outputBuffer bytes.Buffer
			)
			
			testService.httpTestSvrExpectedURI = "/static/~bastion-admin/mycs-test.conf"
			
			// ensure target remotes status is loaded
			// err = testService.target.LoadRemoteRefs()
			// Expect(err).NotTo(HaveOccurred())

			configData, err := vpn.NewVPNConfigData(testService)
			Expect(err).NotTo(HaveOccurred())
			config, err = vpn.NewConfigFromTarget(configData)
			Expect(err).NotTo(HaveOccurred())
			
			tunIfaceName, err = network.GetNextAvailabeInterface("utun")
			Expect(err).NotTo(HaveOccurred())
			Expect(checkDevExists(tunIfaceName)).To(BeFalse())

			client, err = config.NewClient(nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
			Expect(reflect.TypeOf(client).String()).To(Equal("*vpn.wireguard"))

			err = client.Connect()
			Expect(err).NotTo(HaveOccurred())
			Expect(checkDevExists(tunIfaceName)).To(BeTrue())
			
			wgClient, err := wgctrl.New()
			Expect(err).NotTo(HaveOccurred())
			devices, err := wgClient.Devices()
			Expect(err).NotTo(HaveOccurred())
			Expect(len(devices)).To(Equal(1))
			Expect(len((*devices[0]).Peers)).To(Equal(1))

			wgctrlClient, err := vpn.NewWireguardCtrlClient(tunIfaceName)
			Expect(err).NotTo(HaveOccurred())
			status, err := wgctrlClient.StatusText()
			Expect(err).NotTo(HaveOccurred())

			fixResult := regexp.MustCompile(`utun[0-9]+`)
			result := fixResult.ReplaceAllString(status, "utunX")
			Expect(result).To(Equal(deviceStatusOutput))

			// TODO: Fix route check to support linux and windows

			home, _ := homedir.Dir()
			netstat, err := run.NewCLI("/usr/sbin/netstat", home, &outputBuffer, &outputBuffer)
			Expect(err).NotTo(HaveOccurred())
			err = netstat.Run([]string{ "-nrf", "inet" })
			Expect(err).NotTo(HaveOccurred())

			counter := 0
			scanner := bufio.NewScanner(bytes.NewReader(outputBuffer.Bytes()))

			var matchRoutes = func(line string) {
				matched, _ := regexp.MatchString(fmt.Sprintf(`default\s+192.168.111.1\s+UGScg?\s+%s\s+$`, tunIfaceName), line)
				if matched { counter++; return }
				matched, _ = regexp.MatchString(`^34.204.21.102/32\s+([0-9]+\.?)+\s+UGSc\s+en[0-9]\s+$`, line)
				if matched { counter++; return }
				matched, _ = regexp.MatchString(fmt.Sprintf(`^192.168.111.1/32\s+%s\s+USc\s+%s\s+$`, tunIfaceName, tunIfaceName), line)
				if matched { counter++; return }
				matched, _ = regexp.MatchString(fmt.Sprintf(`^192.168.111.194\s+192.168.111.194\s+UH\s+%s\s+$`, tunIfaceName), line)
				if matched { counter++ }
			}

			for scanner.Scan() {
				line := scanner.Text()
				matchRoutes(line)
				logger.DebugMessage("Test route: %s <= %d", line, counter)
			}
			Expect(counter).To(Equal(4))

			// time.Sleep(time.Second * 60)

			err = client.Disconnect()
			Expect(err).NotTo(HaveOccurred())

			// give some time for device shutdown
			time.Sleep(time.Millisecond * 100)
		})
	})
})

func checkDevExists(ifaceName string) bool {
	ifaces, err := net.Interfaces()
	Expect(err).NotTo(HaveOccurred())
	
	for _, i := range ifaces {
		if i.Name == ifaceName {
			return true
		}
	}
	return false
}

const deviceStatusOutput = `interface: utunX (userspace)
  public key: LElaAbWwLh+KE46BOkl9WYvJakalTOYKJXLk2rehUFA=
  private key: (hidden)

peer: AnTKCPYQCkNACBUsB2otfk+V/D3ZiBpNaQJHsSw0hEo=
  endpoint: 34.204.21.102:3399
  allowed ips: 0.0.0.0/0
  latest handshake: 0001-01-01 00:00:00 +0000 UTC
  transfer: 0 B received, 148 B sent
`