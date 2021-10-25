package algo

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type CountingLogPrinter struct {
	LogCalledCnt int
}

func (c *CountingLogPrinter) Out(str string) {
	c.LogCalledCnt++
}

func (c *CountingLogPrinter) Err(str string) {
	c.LogCalledCnt++
}

func (c *CountingLogPrinter) Cmd(str string) {
	c.LogCalledCnt++
}

func (c *CountingLogPrinter) Desc(str string) {
	c.LogCalledCnt++
}

func (c *CountingLogPrinter) Msg(str string) {
	c.LogCalledCnt++
}

var _ = Describe("Installer Algo Tests", func() {
	var (
		installer            *BaseK8sInstaller
		outputBuilderCounter CountingLogPrinter
	)

	const (
		STEPS_NUM = 24
	)

	BeforeEach(func() {
		outputBuilderCounter = *new(CountingLogPrinter)

		ubuntu := Ubuntu_20_4_k8s_1_22{}
		ubuntu.OutputBuilder = &outputBuilderCounter
		ubuntu.BundlePath = ""

		installer = &BaseK8sInstaller{
			K8sStepProvider: &ubuntu,
			OutputBuilder:   &outputBuilderCounter}
	})
	Context("When Installation is executed", func() {
		It("Should count each step", func() {
			installer.Install()
			Expect(outputBuilderCounter.LogCalledCnt).Should(Equal(STEPS_NUM))
		})
	})
	Context("When Uninstallation is executed", func() {
		It("Should count each step", func() {
			installer.Uninstall()
			Expect(outputBuilderCounter.LogCalledCnt).Should(Equal(STEPS_NUM))
		})
	})
})
