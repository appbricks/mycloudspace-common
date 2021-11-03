package vpn_test

import (
	"strings"

	"github.com/appbricks/cloud-builder/target"
	"github.com/appbricks/cloud-builder/terraform"
	"github.com/appbricks/mycloudspace-common/vpn"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	cookbook_mocks "github.com/appbricks/cloud-builder/test/mocks"
	utils_mocks "github.com/mevansam/goutils/test/mocks"
)

var _ = Describe("Client", func() {

	var (
		outputBuffer, errorBuffer strings.Builder

		tgt *target.Target
	)

	Context("create", func() {

		BeforeEach(func() {
			cli := utils_mocks.NewFakeCLI(&outputBuffer, &errorBuffer)			
			tgt = cookbook_mocks.NewMockTarget(cli, "1.1.1.1", 9999, "")

			// ensure target remotes status is loaded
			err := tgt.LoadRemoteRefs()
			Expect(err).NotTo(HaveOccurred())

			output := make(map[string]terraform.Output)
			output["cb_vpc_name"] = terraform.Output{Value: "mycs-test"}
			tgt.Output = &output
		})

		It("does not create a client if the target does not have the correct information", func() {
			configData, err := vpn.NewStaticConfigData(tgt, "bastion-admin", "")
			Expect(err).To(HaveOccurred())
			Expect(configData).To(BeNil())
			Expect(err.Error()).To(Equal("the vpn type was not present in the sandbox build output"))
		})
	})
})
