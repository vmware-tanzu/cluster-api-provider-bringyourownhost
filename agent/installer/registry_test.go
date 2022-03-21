// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// nolint: testpackage
package installer

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Byohost Installer Tests", func() {
	Context("When registry is created", func() {
		type dummyinstaller int

		const (
			dummy122  = dummyinstaller(1221)
			dummy1122 = dummyinstaller(1122)
			dummy124  = dummyinstaller(1243)
		)

		var (
			r registry
		)

		BeforeEach(func() {
			r = newRegistry()
		})

		It("Should be empty", func() {
			osFilters, osBundles := r.ListOS()
			Expect(osFilters).To(HaveLen(0))
			Expect(osBundles).To(HaveLen(0))
			Expect(r.ListK8s("x")).To(HaveLen(0))
			Expect(r.GetInstaller("a", "b")).To(BeNil())
		})
		It("Should allow working with installers", func() {
			Expect(func() { r.AddBundleInstaller("ubuntu", "v1.22.*", dummy122) }).NotTo(Panic())
			Expect(func() { r.AddBundleInstaller("rhel", "v1.22.*", dummy124) }).NotTo(Panic())

			r.AddOsFilter("ubuntu.*", "ubuntu")
			r.AddOsFilter("rhel.*", "rhel")
			r.AddK8sFilter("v1.22.*")

			inst, osBundle := r.GetInstaller("ubuntu-1", "v1.22.1")
			Expect(inst).To(Equal(dummy122))
			Expect(osBundle).To(Equal("ubuntu"))

			osFilters, osBundles := r.ListOS()
			Expect(osFilters).To(ContainElements("rhel.*", "ubuntu.*"))
			Expect(osFilters).To(HaveLen(2))
			Expect(osBundles).To(ContainElements("rhel", "ubuntu"))
			Expect(osBundles).To(HaveLen(2))

			Expect(r.ListK8s("ubuntu")).To(ContainElements("v1.22.*"))
			Expect(r.ListK8s("ubuntu")).To(HaveLen(1))

			Expect(r.ListK8s("rhel")).To(ContainElement("v1.22.*"))
			Expect(r.ListK8s("rhel")).To(HaveLen(1))

			Expect(r.GetInstaller("photon", "v1.22.1")).To(BeNil())
			osFilters, osBundles = r.ListOS()
			Expect(osFilters).To(ContainElements("rhel.*", "ubuntu.*"))
			Expect(osFilters).To(HaveLen(2))
			Expect(osBundles).To(ContainElements("rhel", "ubuntu"))
			Expect(osBundles).To(HaveLen(2))

		})
		It("Should decouple host os from bundle os", func() {
			// Bundle OS does not match filter OS
			r.AddBundleInstaller("UBUNTU", "v1.22.*", dummy122)
			r.AddOsFilter("ubuntu.*", "UBUNTU")
			r.AddK8sFilter("v1.22.*")

			inst, osBundle := r.GetInstaller("ubuntu-1", "v1.22.1")
			Expect(inst).To(Equal(dummy122))
			Expect(osBundle).To(Equal("UBUNTU"))

			// ListOS should return only bundle OS
			_, osBundles := r.ListOS()
			Expect(osBundles).To(ContainElements("UBUNTU"))
			Expect(osBundles).To(HaveLen(1))

			// ListK8s should work with both
			osBundleResult := r.ListK8s("UBUNTU")
			Expect(osBundleResult).To(ContainElements("v1.22.*"))
			Expect(osBundleResult).To(HaveLen(1))

			osHostResult := r.ListK8s("ubuntu-20-04")
			Expect(osHostResult).To(ContainElements("v1.22.*"))
			Expect(osHostResult).To(HaveLen(1))
		})
		It("Should panic on duplicate installers", func() {
			/*
			 * Add is expected to be called with literals only.
			 * Adding a mapping to already existing os and k8s is clearly a typo and bug.
			 * Make it obvious
			 */
			Expect(func() { r.AddBundleInstaller("ubuntu", "v1.22.*", dummy122) }).NotTo(Panic())
			Expect(func() { r.AddBundleInstaller("ubuntu", "v1.22.*", dummy1122) }).To(Panic())
		})
		It("Should not find unsupported K8s versions", func() {
			Expect(func() { r.AddBundleInstaller("ubuntu", "v1.22.*", dummy122) }).NotTo(Panic())
			Expect(func() { r.AddBundleInstaller("rhel", "v1.22.*", dummy122) }).NotTo(Panic())

			// Intentionally skip adding the following filters for unsuported K8s:
			// AddK8sFilter("v1.93.*")
			// AddK8sFilter("v1.94.*")

			inst, osBundle := r.GetInstaller("ubuntu", "v1.93.2")
			Expect(inst).To(BeNil())
			Expect(osBundle).To(Equal(""))

			inst, osBundle = r.GetInstaller("rhel", "v1.94.3")
			Expect(inst).To(BeNil())
			Expect(osBundle).To(Equal(""))
		})
	})
})
