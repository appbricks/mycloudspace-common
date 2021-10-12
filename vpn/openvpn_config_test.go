package vpn_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("OpenVPN Config", func() {

	var (
		err error
	)

	Context("load", func() {

		It("load openvpn vpn config to connect to a target", func() {
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
