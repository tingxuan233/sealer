package test

import (
	"fmt"
	"github.com/alibaba/sealer/test/testhelper"
	"github.com/alibaba/sealer/test/testhelper/settings"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("sealer inspect", func() {
	imageName := "kubernetes:v1.19.9"
	Context("Yaml images", func() {
		It("inspect cluster", func() {
			sess, err := testhelper.Start(fmt.Sprintf("sealer inspect %s", imageName))
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess, settings.MaxWaiteTime).Should(Exit(0))
		})
	})
	Context("Cluster images", func() {
		It("inspect cluster -c", func() {
			sess, err := testhelper.Start(fmt.Sprintf("sealer inspect -c %s", imageName))
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess, settings.MaxWaiteTime).Should(Exit(0))
		})
	})
})
