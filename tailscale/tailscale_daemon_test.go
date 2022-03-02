package tailscale_test

import (
	"strings"

	"github.com/appbricks/mycloudspace-common/tailscale"
	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/run"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Tailscale Daemon", func() {

	It("start the tailscale daemon and validate it starts up without errors", func() {
		isAdmin, err := run.IsAdmin()
		Expect(err).NotTo(HaveOccurred())
		if !isAdmin {
			Fail("This test needs to be run with root privileges. i.e. sudo -E go test -v ./...")
		}

		var (
			outputBuffer strings.Builder
		)

		tsd := tailscale.NewTailscaleDaemon(tmpDir, &outputBuffer)
		err = tsd.Start()
		Expect(err).ToNot(HaveOccurred())
		tsd.Stop()

		output := outputBuffer.String()
		logger.DebugMessage("Tailscale Daemon log: \n%s\n", output)
		Expect(output).To(ContainSubstring("logtail started"))
		Expect(output).To(ContainSubstring("Program starting:"))
		Expect(output).To(ContainSubstring("LogID:"))
		Expect(output).To(ContainSubstring("logpolicy:"))
		Expect(output).To(ContainSubstring("flushing log."))
		Expect(output).To(ContainSubstring("logger closing down"))
	})
})
