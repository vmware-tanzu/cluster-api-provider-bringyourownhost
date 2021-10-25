package algo

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type LogPrinterCounter struct {
	LogCalledCnt int
}

func (c *LogPrinterCounter) Out(str string) {
	c.LogCalledCnt++
}

func (c *LogPrinterCounter) Err(str string) {
	c.LogCalledCnt++
}

func (c *LogPrinterCounter) Cmd(str string) {
	c.LogCalledCnt++
}

func (c *LogPrinterCounter) Desc(str string) {
	c.LogCalledCnt++
}

func (c *LogPrinterCounter) Msg(str string) {
	c.LogCalledCnt++
}

var _ = Describe("Installer Algo Tests", func() {
	var (
		installer         *BaseK8sInstaller
		logPrinterCounter LogPrinterCounter
	)

	const (
		STEPS_NUM = 24
	)

	BeforeEach(func() {
		/*
			Initialize a new log printer counter each time a
			context is started to be used as a standard output device/pipe.

			Also initialize a new installer and set this
			log printer counter as its default logging system.

			The test will count the number of logged steps performed by the
			installer during installation/uninstallation and compare
			the value with the expected steps count.
		*/

		logPrinterCounter = LogPrinterCounter{}

		ubuntu := Ubuntu_20_4_k8s_1_22{}
		ubuntu.OutputBuilder = &logPrinterCounter
		ubuntu.BundlePath = ""

		installer = &BaseK8sInstaller{
			K8sStepProvider: &ubuntu,
			OutputBuilder:   &logPrinterCounter}
	})
	Context("When Installation is executed", func() {
		It("Should count each step", func() {
			err := installer.Install()
			Expect(err).ShouldNot((HaveOccurred()))
			Expect(logPrinterCounter.LogCalledCnt).Should(Equal(STEPS_NUM))
		})
	})
	Context("When Uninstallation is executed", func() {
		It("Should count each step", func() {
			err := installer.Uninstall()
			Expect(err).ShouldNot((HaveOccurred()))
			Expect(logPrinterCounter.LogCalledCnt).Should(Equal(STEPS_NUM))
		})
	})
})
