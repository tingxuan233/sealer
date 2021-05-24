package test

import (
	"fmt"
	"github.com/alibaba/sealer/test/testhelper"

	"github.com/alibaba/sealer/test/testhelper/settings"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("sealer images", func() {
	Context("with roofs images", func() {
		It("output images", func() {
			sess, err := testhelper.Start(fmt.Sprintf("sealer images "))
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess, settings.MaxWaiteTime).Should(Exit(0))
		})
	})
})
