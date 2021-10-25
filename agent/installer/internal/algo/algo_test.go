package algo

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type OutputBuilderCounter struct {
	LogCalledCnt int
}

func (c *OutputBuilderCounter) Out(str string) {
	c.LogCalledCnt++
}

func (c *OutputBuilderCounter) Err(str string) {
	c.LogCalledCnt++
}

func (c *OutputBuilderCounter) Cmd(str string) {
	c.LogCalledCnt++
}

func (c *OutputBuilderCounter) Desc(str string) {
	c.LogCalledCnt++
}

func (c *OutputBuilderCounter) Msg(str string) {
	c.LogCalledCnt++
}

var _ = Describe("Installer Algo Tests", func() {
	var (
		installer            *BaseK8sInstaller
		outputBuilderCounter OutputBuilderCounter
	)

	const (
		stepsNum = 24
	)

	BeforeEach(func() {
		outputBuilderCounter = OutputBuilderCounter{}

		ubuntu := Ubuntu20_4K8s1_22{}
		ubuntu.OutputBuilder = &outputBuilderCounter
		ubuntu.BundlePath = ""

		installer = &BaseK8sInstaller{
			K8sStepProvider: &ubuntu,
			OutputBuilder:   &outputBuilderCounter}
	})
	Context("When Installation is executed", func() {
		It("Should count each step", func() {
			err := installer.Install()
			Expect(err).ShouldNot((HaveOccurred()))
			Expect(outputBuilderCounter.LogCalledCnt).Should(Equal(stepsNum))
		})
	})
	Context("When Uninstallation is executed", func() {
		It("Should count each step", func() {
			err := installer.Uninstall()
			Expect(err).ShouldNot((HaveOccurred()))
			Expect(outputBuilderCounter.LogCalledCnt).Should(Equal(stepsNum))
		})
	})
})
