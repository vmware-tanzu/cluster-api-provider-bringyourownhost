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
                It("Should be empty", func() {
			r := NewRegistry()

			Expect(len(r.ListOS())).To(Equal(0))
			Expect(len(r.ListK8s("x"))).To(Equal(0))
			Expect(r.GetInstaller("a","b")).To(BeNil())
                })
	        It("Should allow adding and gettnig installers", func() {
			r := NewRegistry()
			Expect(r.Add("ubuntu", "1.22", 122)).ShouldNot((HaveOccurred()))
			Expect(r.Add("ubuntu", "1.22", 122)).Should((HaveOccurred()))
			Expect(r.Add("ubuntu", "1.23", 123)).ShouldNot((HaveOccurred()))
			Expect(r.Add("rhel", "1.24", 124)).ShouldNot((HaveOccurred()))

			Expect(r.GetInstaller("ubuntu","1.22")).To(Equal(122))
			Expect(r.GetInstaller("ubuntu","1.23")).To(Equal(123))
			Expect(r.GetInstaller("rhel","1.24")).To(Equal(124))

			osList := r.ListOS()
			sort.Strings(osList)
			Expect(osList).To(BeEquivalentTo([]string{"rhel", "ubuntu"}))

			k8sList := r.ListK8s("ubuntu")
			sort.Strings(k8sList)
			Expect(k8sList).To(BeEquivalentTo([]string{"1.22", "1.23"}))

			Expect(r.ListK8s("rhel")).To(BeEquivalentTo([]string{"1.24"}))

			Expect(r.GetInstaller("photon","1.22")).To(BeNil())
			osList = r.ListOS()
			sort.Strings(osList)
			Expect(osList).To(BeEquivalentTo([]string{"rhel", "ubuntu"}))
	        })
	})

})
