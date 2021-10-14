// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package installer

import (
	"sort"
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
			r = NewRegistry()
		})

                It("Should be empty", func() {
			Expect(r.ListOS()).To(HaveLen(0))
			Expect(r.ListK8s("x")).To(HaveLen(0))
			Expect(r.GetInstaller("a","b")).To(BeNil())
                })
	        It("Should allow working with installers", func() {
			Expect(func() { r.Add("ubuntu", "1.22", dummy122) }).NotTo(Panic())
			Expect(func() { r.Add("ubuntu", "1.23", dummy123) }).NotTo(Panic())
			Expect(func() { r.Add("rhel", "1.24", dummy124) }).NotTo(Panic())

			Expect(r.GetInstaller("ubuntu","1.22")).To(Equal(dummy122))
			Expect(r.GetInstaller("ubuntu","1.23")).To(Equal(dummy123))
			Expect(r.GetInstaller("rhel","1.24")).To(Equal(dummy124))

			Expect(r.ListOS()).To(ContainElements("rhel", "ubuntu"))
			Expect(r.ListOS()).To(HaveLen(2))

			Expect(r.ListK8s("ubuntu")).To(ContainElements("1.22", "1.23"))
			Expect(r.ListK8s("ubuntu")).To(HaveLen(2))

			Expect(r.ListK8s("rhel")).To(ContainElement("1.24"))
			Expect(r.ListK8s("rhel")).To(HaveLen(1))

			Expect(r.GetInstaller("photon","1.22")).To(BeNil())
			osList := r.ListOS()
			sort.Strings(osList)
			Expect(r.ListOS()).To(ContainElements("rhel", "ubuntu"))
			Expect(r.ListOS()).To(HaveLen(2))

	        })
		It("Should panic on duplicate installers", func() {
			/*
			 * Add is expected to be called with literals only.
			 * Adding a mapping to already existing os and k8s is clearly a typo and bug.
			 * Make it obvious
			 */
			Expect(func() { r.Add("ubuntu", "1.22", dummy122) }).NotTo(Panic())
			Expect(func() { r.Add("ubuntu", "1.22", dummy1122) }).To(Panic())
		})
	})
})
