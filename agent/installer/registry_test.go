// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package installer

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Byohost Installer Tests", func() {
	Context("When registry is created", func() {
		type dummyinstaller int

		const (
			dummy122  = dummyinstaller(122)
			dummy1122 = dummyinstaller(1122)
			dummy123  = dummyinstaller(123)
			dummy124  = dummyinstaller(124)
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
			Expect(func() { r.AddBundleInstaller("ubuntu", "1.22", dummy122) }).NotTo(Panic())
			Expect(func() { r.AddBundleInstaller("ubuntu", "1.23", dummy123) }).NotTo(Panic())
			Expect(func() { r.AddBundleInstaller("rhel", "1.24", dummy124) }).NotTo(Panic())
			r.AddOsFilter("ubuntu.*", "ubuntu")
			r.AddOsFilter("rhel.*", "rhel")

			Expect(r.GetInstaller("ubuntu-1", "1.22")).To(Equal(dummy122))
			Expect(r.GetInstaller("ubuntu-2", "1.23")).To(Equal(dummy123))
			Expect(r.GetInstaller("rhel-1", "1.24")).To(Equal(dummy124))

			osFilters, osBundles := r.ListOS()
			Expect(osFilters).To(ContainElements("rhel.*", "ubuntu.*"))
			Expect(osFilters).To(HaveLen(2))
			Expect(osBundles).To(ContainElements("rhel", "ubuntu"))
			Expect(osBundles).To(HaveLen(2))

			Expect(r.ListK8s("ubuntu")).To(ContainElements("1.22", "1.23"))
			Expect(r.ListK8s("ubuntu")).To(HaveLen(2))

			Expect(r.ListK8s("rhel")).To(ContainElement("1.24"))
			Expect(r.ListK8s("rhel")).To(HaveLen(1))

			Expect(r.GetInstaller("photon", "1.22")).To(BeNil())
			osFilters, osBundles = r.ListOS()
			Expect(osFilters).To(ContainElements("rhel.*", "ubuntu.*"))
			Expect(osFilters).To(HaveLen(2))
			Expect(osBundles).To(ContainElements("rhel", "ubuntu"))
			Expect(osBundles).To(HaveLen(2))

		})
		It("Should panic on duplicate installers", func() {
			/*
			 * Add is expected to be called with literals only.
			 * Adding a mapping to already existing os and k8s is clearly a typo and bug.
			 * Make it obvious
			 */
			Expect(func() { r.AddBundleInstaller("ubuntu", "1.22", dummy122) }).NotTo(Panic())
			Expect(func() { r.AddBundleInstaller("ubuntu", "1.22", dummy1122) }).To(Panic())
		})
	})
})
