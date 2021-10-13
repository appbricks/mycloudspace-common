package tailscale_test

import (
	"strings"

	"github.com/appbricks/mycloudspace-common/tailscale"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Tailscale Daemon", func() {

	var (
		err error
	)

	It("start the tailscale daemon and validate it starts up without errors", func() {

		var (
			outputBuffer strings.Builder
		)

		tsd := tailscale.NewTailscaleDaemon(tmpDir, &outputBuffer)
		err = tsd.Start()
		Expect(err).ToNot(HaveOccurred())
		tsd.Stop()
		Expect(outputBuffer.String()).To(ContainSubstring("Program starting:"))
		Expect(outputBuffer.String()).To(ContainSubstring("LogID:"))
		Expect(outputBuffer.String()).To(ContainSubstring("flushing log."))
		Expect(outputBuffer.String()).To(ContainSubstring("logger closing down"))
		Expect(outputBuffer.String()).To(ContainSubstring("logtail: dialed"))
	})
})
