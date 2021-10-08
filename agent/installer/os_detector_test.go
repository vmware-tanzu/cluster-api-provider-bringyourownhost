package installer

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func (osd *osDetector) mockHostSystemInfo(os, ver, arch string) (string, error) {
	out := "  Static hostname: ubuntu\n" +
		"        Icon name: computer-vm\n" +
		"          Chassis: vm\n" +
		"       Machine ID: 242642b0e734472abaf8c5337e1174c4\n" +
		"          Boot ID: 181f08d651b76h39be5b138231427c5c\n" +
		"   Virtualization: vmware\n" +
		" Operating System: " + os + " " + ver + " LTS\n" +
		"           Kernel: Linux 5.11.0-27-generic\n" +
		"     Architecture: " + arch + "\n"

	return out, nil
}

func (osd *osDetector) mockHostSystemInfoCustomString(str string) (string, error) {
	return str, nil
}

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
		d = &osDetector{}
		os = "Ubuntu"
		ver = "20.04.3"
		arch = "x86-64"
	})

	Context("When the OS is detected", func() {
		It("Should return string in normalized format and cache it", func() {
			detectedOS, err = d.delegateDetect(func() (string, error) { return d.mockHostSystemInfo(os, ver, arch) })
			expectedDetectedOS := "Ubuntu_20.04.3_x86-64"
			Expect(err).ShouldNot((HaveOccurred()))
			Expect(detectedOS).To(Equal(expectedDetectedOS))
			Expect(d.cachedNormalizedOS).To(Equal(expectedDetectedOS))
		})

		It("Should return string in normalized format and work with OS names with more than one word", func() {
			os = "Red Hat Enterprise Linux"
			ver = "8.1"
			expectedDetectedOS := "Red_Hat_Enterprise_Linux_8.1_x86-64"
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

		It("Should return error if output is missing", func() {
			_, err = d.delegateDetect(func() (string, error) {
				return d.mockHostSystemInfoCustomString("")
			})
			Expect(err).Should((HaveOccurred()))
		})

		It("Should return error if output is random string", func() {
			_, err = d.delegateDetect(func() (string, error) {
				return d.mockHostSystemInfoCustomString("wef9sdf092g\nd2g39\n\n\nd92faad")
			})
			Expect(err).Should((HaveOccurred()))
		})
	})

})
