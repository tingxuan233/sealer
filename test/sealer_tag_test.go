package test

import (
	"fmt"
	"github.com/alibaba/sealer/test/testhelper"
	"github.com/alibaba/sealer/test/testhelper/settings"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("sealer tag", func() {
	Context("start tag", func() {
		Context("with roofs images", func() {
			oldName := "kubernetes:v1.19.9"
			newName := "registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.9"
			AfterEach(func() {
				sess, err := testhelper.Start(fmt.Sprintf("sealer images "))
				Expect(err).NotTo(HaveOccurred())
				Eventually(sess, settings.MaxWaiteTime).Should(Exit(0))
			})
			It("tag cluster", func() {
				sess, err := testhelper.Start(fmt.Sprintf("sealer tag %s %s", oldName, newName))
				Expect(err).NotTo(HaveOccurred())
				Eventually(sess, settings.MaxWaiteTime).Should(Exit(0))
			})
		})
	})
})
