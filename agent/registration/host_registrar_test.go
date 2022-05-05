// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// nolint: testpackage
package registration

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"
)

func getMockFile(targetOs string) ([]byte, error) {
	out := fmt.Sprintf(`NAME="Ubuntu"
VERSION="20.04.4 LTS (Focal Fossa)"
ID=ubuntu
ID_LIKE=debian
PRETTY_NAME="%s"
VERSION_ID="20.04"
HOME_URL="https://www.ubuntu.com/"
SUPPORT_URL="https://help.ubuntu.com/"
BUG_REPORT_URL="https://bugs.launchpad.net/ubuntu/"
PRIVACY_POLICY_URL="https://www.ubuntu.com/legal/terms-and-policies/privacy-policy"
VERSION_CODENAME=focal
UBUNTU_CODENAME=focal`, targetOs)
	return []byte(out), nil
}

var _ = Describe("Host Registrar Tests", func() {
	Context("When the OS is detected", func() {
		It("Should return the operating system for os following /etc/os-release", func() {
			targetOs := "Ubuntu 20.04.4 LTS"
			detectedOS, err := getOperatingSystem(func(string) ([]byte, error) { return getMockFile(targetOs) })
			Expect(err).ShouldNot(HaveOccurred())
			Expect(detectedOS).To(Equal("Ubuntu 20.04.4 LTS"))
		})

		It("Should return the operating system for os following /usr/lib/os-release", func() {
			targetOs := "Clear Linux Initramfs"
			detectedOS, err := getOperatingSystem(func(releaseFile string) ([]byte, error) {
				if releaseFile == "/etc/os-release" {
					return nil, os.ErrNotExist
				}
				return getMockFile(targetOs)
			})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(detectedOS).To(Equal("Clear Linux Initramfs"))
		})

		It("Should not error with real hostnamectl", func() {
			_, err := getOperatingSystem(ioutil.ReadFile)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})

	Context("When the os-release file is missing", func() {
		It("Should return error", func() {
			_, err := getOperatingSystem(func(string) ([]byte, error) {
				return nil, os.ErrNotExist
			})
			Expect(err.Error()).To(Equal("error opening file : file does not exist"))
		})
	})

	Context("When the os-release does not contain PRETTY_NAME", func() {
		It("Should return Unknown as operating system", func() {
			detectedOS, err := getOperatingSystem(func(string) ([]byte, error) { return []byte("some_file_without_PRETTY_NAME"), nil })
			Expect(err).ShouldNot(HaveOccurred())
			Expect(detectedOS).To(Equal("Unknown"))
		})
	})
})
