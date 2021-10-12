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
		err error

		outputBuffer, errorBuffer strings.Builder

		cli *utils_mocks.FakeCLI		
		tgt *target.Target

		output map[string]terraform.Output

		config vpn.Config
	)

	Context("create", func() {

		BeforeEach(func() {
			cli = utils_mocks.NewFakeCLI(&outputBuffer, &errorBuffer)			
			tgt = cookbook_mocks.NewMockTarget(cli, "1.1.1.1", 9999, "")

			output = make(map[string]terraform.Output)
			tgt.Output = &output
		})

		It("does not create a client if the target does not have the correct information", func() {
			config, err = vpn.NewConfigFromTarget(tgt, "bastion-admin", "")
			Expect(err).To(HaveOccurred())
			Expect(config).To(BeNil())
			Expect(err.Error()).To(Equal("target \"fakeRecipe/fakeIAAS/\" is not a bastion node"))

			tgt.Recipe.(*cookbook_mocks.FakeRecipe).SetBastion()
			config, err = vpn.NewConfigFromTarget(tgt, "bastion-admin", "")
			Expect(err).To(HaveOccurred())
			Expect(config).To(BeNil())
			Expect(err.Error()).To(Equal("target vpn type \"\" is not supported"))
		})
	})
})
