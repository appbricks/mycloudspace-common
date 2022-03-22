package vpn_test

import (
	"fmt"
	"os"
	"reflect"

	"github.com/appbricks/mycloudspace-common/vpn"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Wireguard Config", func() {

	var (
		err error

		testService *testService
		config     vpn.Config

		downloadPath string
	)

	Context("load", func() {

		BeforeEach(func() {
			// test http server to mock bastion HTTPS 
			// backend for vpn config retrieval
			testService = startTestService()

			downloadPath, err = os.MkdirTemp("", "vpn");
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			testService.stop()
			os.RemoveAll(downloadPath)
		})

		It("load wireguard vpn config to connect to a target", func() {
			testService.httpTestSvrExpectedURI = "/static/~bastion-admin/mycs-test.conf"

			configData, err := vpn.NewVPNConfigData(testService)
			Expect(err).NotTo(HaveOccurred())
			config, err = vpn.NewConfigFromTarget(configData)
			Expect(err).NotTo(HaveOccurred())
			Expect(config).ToNot(BeNil())
			Expect(reflect.TypeOf(config).String()).To(Equal("*vpn.wireguardConfig"))		
			Expect(config.Config()).To(Equal(wireguardConfig))
			Expect(testService.httpTestSvrErr).NotTo(HaveOccurred())		
			
			desc, err := config.Save(downloadPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(desc[0:400]).To(MatchRegexp(fmt.Sprintf(wireguardConfigSave, downloadPath)))
		})
	})
})
