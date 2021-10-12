package vpn_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("OpenVPN Client", func() {

	var (
		err error
	)

	Context("create", func() {

		It("create openvpn vpn client to connect to a target", func() {
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
