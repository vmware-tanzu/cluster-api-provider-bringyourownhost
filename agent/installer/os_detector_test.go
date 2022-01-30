// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// nolint: testpackage
package installer

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type MockHostnamectl struct {
	callCount int
}

func (mh *MockHostnamectl) Get(os, ver, arch string) (string, error) {
	mh.callCount++
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

var _ = Describe("Byohost Installer Tests", func() {

	var (
		d          *osDetector
		mh         *MockHostnamectl
		os         string
		ver        string
		arch       string
		detectedOS string
		err        error
	)

	BeforeEach(func() {
		d = &osDetector{}
		mh = &MockHostnamectl{}
		os = "Ubuntu"
		ver = "20.04.3"
		arch = "x86-64"
	})

	Context("When the OS is detected", func() {
		It("Should return string in normalized format", func() {
			detectedOS, err = d.DetectByHostnamectl(func() (string, error) { return mh.Get(os, ver, arch) })
			Expect(err).ShouldNot(HaveOccurred())
			Expect(detectedOS).To(Equal("Ubuntu_20.04.3_x86-64"))
		})
		It("Should cache OS and not execute again getHostnamectl", func() {
			_, err = d.DetectByHostnamectl(func() (string, error) { return mh.Get(os, ver, arch) })
			Expect(err).ShouldNot(HaveOccurred())
			Expect(mh.callCount).To(Equal(1))
			_, err = d.DetectByHostnamectl(func() (string, error) { return mh.Get(os, ver, arch) })
			Expect(err).ShouldNot(HaveOccurred())
			Expect(mh.callCount).To(Equal(1))
		})

		It("Should return string in normalized format and work with OS names with more than one word", func() {
			os = "Red Hat Enterprise Linux"
			ver = "8.1"
			detectedOS, err = d.DetectByHostnamectl(func() (string, error) { return mh.Get(os, ver, arch) })
			Expect(err).ShouldNot(HaveOccurred())
			Expect(detectedOS).To(Equal("Red_Hat_Enterprise_Linux_8.1_x86-64"))
		})

		It("Should not error with real hostnamectl", func() {
			_, err = d.Detect()
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
	Context("When the OS is not detected", func() {
		It("Should return error if OS distribution is missing", func() {
			os = ""
			_, err = d.DetectByHostnamectl(func() (string, error) { return mh.Get(os, ver, arch) })
			Expect(err).Should(HaveOccurred())
		})

		It("Should return error if OS version is missing", func() {
			ver = ""
			_, err = d.DetectByHostnamectl(func() (string, error) { return mh.Get(os, ver, arch) })
			Expect(err).Should(HaveOccurred())
		})

		It("Should return error if OS architecture is missing", func() {
			arch = ""
			_, err = d.DetectByHostnamectl(func() (string, error) { return mh.Get(os, ver, arch) })
			Expect(err).Should(HaveOccurred())
		})

		It("Should return error if output is missing", func() {
			_, err = d.DetectByHostnamectl(func() (string, error) {
				return "", nil
			})
			Expect(err).Should(HaveOccurred())
		})

		It("Should return error if output is random string", func() {
			_, err = d.DetectByHostnamectl(func() (string, error) {
				return "wef9sdf092g\nd2g39\n\n\nd92faad", nil
			})
			Expect(err).Should(HaveOccurred())
		})
	})

})
