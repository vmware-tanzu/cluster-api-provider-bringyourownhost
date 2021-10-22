package algo

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type ConsoleLogPrinter struct {
	LogCalledCnt int
}

func (c *ConsoleLogPrinter) Out(str string) {
	c.LogCalledCnt++
}

func (c *ConsoleLogPrinter) Err(str string) {
	c.LogCalledCnt++
}

func (c *ConsoleLogPrinter) Cmd(str string) {
	c.LogCalledCnt++
}

func (c *ConsoleLogPrinter) Desc(str string) {
	c.LogCalledCnt++
}

func (c *ConsoleLogPrinter) Msg(str string) {
	c.LogCalledCnt++
}

var _ = Describe("Installer Algo Tests", func() {
	var (
		installer     *BaseK8sInstaller
		outputBuilder ConsoleLogPrinter
	)

	const (
		INSTALL_CNT = 24
	)

	BeforeEach(func() {
		ubuntu := Ubuntu_20_4_k8s_1_22{}
		ubuntu.OutputBuilder = &outputBuilder
		ubuntu.BundlePath = ""

		installer = &BaseK8sInstaller{
			K8sStepProvider: &ubuntu,
			OutputBuilder:   &outputBuilder}
	})
	Context("When Installation is executed", func() {
		It("Should print each step", func() {
			installer.Install()
			Expect(outputBuilder.LogCalledCnt).Should(Equal(INSTALL_CNT))
		})
	})
	Context("When Uninstallation is executed", func() {
		It("Should print each step", func() {
			installer.Uninstall()
			Expect(outputBuilder.LogCalledCnt).Should(Equal(INSTALL_CNT * 2))
		})
	})
})
