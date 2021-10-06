package installer

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Byohost Installer Tests", func() {

	var (
		d          *osDetector
		os         string
		ver        string
		arch       string
		detectedOS string
		err        error
	)

	BeforeEach(func() {
		d = newOSDetector()
		os = "Ubuntu"
		ver = "20.04.3"
		arch = "x64"
	})

	Context("When the OS is detected", func() {
		It("Should return string in normalized format", func() {
			expectedDetectedOS := os + "_" + ver + "_" + arch
			detectedOS, err = d.delegateDetect(func() (string, error) { return d.mockHostSystemInfo(os, ver, arch) })
			Expect(err).ShouldNot((HaveOccurred()))
			Expect(detectedOS).To(Equal(expectedDetectedOS))
		})

		It("Should return string in normalized format and work with OS names with more than one word", func() {
			os = "Red Hat Enterprise Linux"
			ver = "8.1"
			expectedDetectedOS := strings.ReplaceAll(os+"_"+ver+"_"+arch, " ", "_")
			detectedOS, err = d.delegateDetect(func() (string, error) { return d.mockHostSystemInfo(os, ver, arch) })
			Expect(err).ShouldNot((HaveOccurred()))
			Expect(detectedOS).To(Equal(expectedDetectedOS))
		})
	})
	Context("When the OS is not detected", func() {
		It("Should return error if OS distribution is missing", func() {
			os = ""
			_, err = d.delegateDetect(func() (string, error) { return d.mockHostSystemInfo(os, ver, arch) })
			Expect(err).Should((HaveOccurred()))
		})

		It("Should return error if OS version is missing", func() {
			ver = ""
			_, err = d.delegateDetect(func() (string, error) { return d.mockHostSystemInfo(os, ver, arch) })
			Expect(err).Should((HaveOccurred()))
		})

		It("Should return error if OS architecture is missing", func() {
			arch = ""
			_, err = d.delegateDetect(func() (string, error) { return d.mockHostSystemInfo(os, ver, arch) })
			Expect(err).Should((HaveOccurred()))
		})
	})

})
