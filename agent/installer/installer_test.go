// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// nolint: testpackage
package installer

import (
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/installer/internal/algo"
)

var _ = Describe("Byohost Installer Tests", func() {
	const testTag = "test-tag"

	Context("When installer is created for unsupported OS", func() {
		It("Should return error", func() {
			_, err := newUnchecked("Ubuntu_99.04.3_x86-64", "k8s", "", logr.Discard(), nil)
			Expect(err).Should(HaveOccurred())
		})
	})
	Context("When installer is created with empty download path", func() {
		It("Should return error", func() {
			_, err := New("", "k8s", logr.Discard())
			Expect(err).Should(HaveOccurred())
		})
	})
	Context("When installer is created", func() {
		It("Install/uninstall should return error for unsupported k8s", func() {

			// Currently supported versions: v1.21.*, v1.22.*, v1.23.*
			unsupportedMinorVer := "v1.20.1"
			unsupportedMajorVer := "v0.21.1"

			_, osList := ListSupportedOS()
			for _, os := range osList {
				i := NewPreviewInstaller(os, nil)

				err := i.Install("", "unsupported-k8s", testTag)
				Expect(err).Should(HaveOccurred())

				err = i.Uninstall("", "unsupported-k8s", testTag)
				Expect(err).Should(HaveOccurred())

				err = i.Install("", unsupportedMinorVer, testTag)
				Expect(err).Should(HaveOccurred())

				err = i.Uninstall("", unsupportedMinorVer, testTag)
				Expect(err).Should(HaveOccurred())

				err = i.Install("", unsupportedMajorVer, testTag)
				Expect(err).Should(HaveOccurred())

				err = i.Uninstall("", unsupportedMajorVer, testTag)
				Expect(err).Should(HaveOccurred())
			}
		})
	})
	Context("When installer is created", func() {
		It("Install/uninstall should call only the output builder", func() {
			_, osList := ListSupportedOS()
			for _, os := range osList {
				for _, k8s := range ListSupportedK8s(os) {
					{
						ob := algo.OutputBuilderCounter{}
						i := NewPreviewInstaller(os, &ob)
						err := i.Install("", k8s, testTag)
						Expect(err).ShouldNot(HaveOccurred())
						Expect(ob.LogCalledCnt).Should(Equal(22))
					}

					{
						ob := algo.OutputBuilderCounter{}
						i := NewPreviewInstaller(os, &ob)
						err := i.Uninstall("", k8s, testTag)
						Expect(err).ShouldNot(HaveOccurred())
						Expect(ob.LogCalledCnt).Should(Equal(22))
					}
				}
			}
		})
	})
	Context("When ListSupportedOS is called", func() {
		It("Should return non-empty result", func() {
			_, osList := ListSupportedOS()
			Expect(osList).ShouldNot(BeEmpty())
		})
	})
	Context("When ListSupportedK8s is called for all supported bundle OSes", func() {
		It("Should return non-empty result", func() {
			_, osList := ListSupportedOS()
			for _, osBundle := range osList {
				Expect(ListSupportedK8s(osBundle)).ShouldNot(BeEmpty())
			}
		})
	})
	Context("When ListSupportedK8s is called for supported host OS", func() {
		It("Should return non-empty result", func() {
			Expect(ListSupportedK8s("Ubuntu_20.04.3_x86-64")).ShouldNot(BeEmpty())
		})
	})
	Context("When PreviewChanges is called for all supported os and k8s", func() {
		It("Should not return error", func() {
			_, osList := ListSupportedOS()
			for _, os := range osList {
				for _, k8s := range ListSupportedK8s(os) {
					_, _, err := PreviewChanges(os, k8s)
					Expect(err).ShouldNot(HaveOccurred())
				}
			}
		})
	})
	Context("When PreviewChanges is called for supported os and k8s", func() {
		It("Should return non-empty result", func() {
			_, osList := ListSupportedOS()
			os := osList[0]
			k8s := ListSupportedK8s(os)[0]
			install, uninstall, err := PreviewChanges(os, k8s)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(install).Should(ContainSubstring("Installing"))
			Expect(install).ShouldNot(ContainSubstring("Uninstalling"))
			Expect(uninstall).Should(ContainSubstring("Uninstalling"))
			Expect(uninstall).ShouldNot(ContainSubstring("Installing"))
		})
	})
	Context("When PreviewChanges is called for non-supported os and k8s", func() {
		It("Should return error", func() {
			os := "a"
			k8s := "a"
			_, _, err := PreviewChanges(os, k8s)
			Expect(err).Should(HaveOccurred())
			Expect(err).Should(Equal(ErrOsK8sNotSupported))
		})
	})
	Context("When installer is created", func() {
		It("Should be possible to do so using host os or bundle os ", func() {
			Expect(func() { NewPreviewInstaller("Ubuntu_20.04.1_x86-64", nil) }).NotTo(Panic())
			Expect(func() { NewPreviewInstaller("Ubuntu_20.04.3_x86-64", nil) }).NotTo(Panic())
		})
	})
})

func NewPreviewInstaller(os string, ob algo.OutputBuilder) *installer {
	i, err := newUnchecked(os, "k8s", "", logr.Discard(), ob)
	if err != nil {
		panic(err)
	}
	return i
}
