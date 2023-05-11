// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package installer

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Byohost Installer Tests", func() {
	Context("When registry is created", func() {
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
		})
		It("Should allow working with installers", func() {
			Expect(func() { r.AddBundleInstaller("ubuntu", "v1.22.*") }).NotTo(Panic())
			Expect(func() { r.AddBundleInstaller("rhel", "v1.22.*") }).NotTo(Panic())

			r.AddOsFilter("ubuntu.*", "ubuntu")
			r.AddOsFilter("rhel.*", "rhel")
			r.AddK8sFilter("v1.22.*")

			osBundle := r.ResolveOsToOsBundle("ubuntu-1")
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

			Expect(r.ResolveOsToOsBundle("photon")).To(Equal(""))
			osFilters, osBundles = r.ListOS()
			Expect(osFilters).To(ContainElements("rhel.*", "ubuntu.*"))
			Expect(osFilters).To(HaveLen(2))
			Expect(osBundles).To(ContainElements("rhel", "ubuntu"))
			Expect(osBundles).To(HaveLen(2))

		})
		It("Should decouple host os from bundle os", func() {
			// Bundle OS does not match filter OS
			r.AddBundleInstaller("UBUNTU", "v1.22.*")
			r.AddOsFilter("ubuntu.*", "UBUNTU")
			r.AddK8sFilter("v1.22.*")

			osBundle := r.ResolveOsToOsBundle("ubuntu-1")
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
			Expect(func() { r.AddBundleInstaller("ubuntu", "v1.22.*") }).NotTo(Panic())
			Expect(func() { r.AddBundleInstaller("ubuntu", "v1.22.*") }).To(Panic())
		})
		It("Should not find unsupported K8s versions", func() {
			Expect(func() { r.AddBundleInstaller("ubuntu", "v1.22.*") }).NotTo(Panic())
			Expect(func() { r.AddBundleInstaller("rhel", "v1.22.*") }).NotTo(Panic())

			// Intentionally skip adding the following filters for unsuported K8s:
			// AddK8sFilter("v1.93.*")
			// AddK8sFilter("v1.94.*")

			osBundle := r.ResolveOsToOsBundle("ubuntu")
			Expect(osBundle).To(Equal(""))

			osBundle = r.ResolveOsToOsBundle("rhel")
			Expect(osBundle).To(Equal(""))
		})
	})

	Context("When supported registry is fetched", func() {

		r := GetSupportedRegistry()

		It("Should match with the supported os and k8s versions", func() {
			osFilters, osBundles := r.ListOS()
			Expect(osFilters).To(ContainElements("Ubuntu_20.04.*_x86-64"))
			Expect(osFilters).To(HaveLen(1))
			Expect(osBundles).To(ContainElements("Ubuntu_20.04.1_x86-64"))
			Expect(osBundles).To(HaveLen(1))

			osBundleResult := r.ListK8s("Ubuntu_20.04.1_x86-64")
			Expect(osBundleResult).To(ContainElements("v1.22.*", "v1.23.*", "v1.24.*","v1.25.*","v1.26.*"))
			Expect(osBundleResult).To(HaveLen(5))
		})
	})
})
