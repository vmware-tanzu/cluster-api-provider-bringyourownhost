package installer

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Byohost Installer Tests", func() {
	 Context("When registry is created", func() {
                It("Should be empty", func() {
			r := NewRegistry()

			Expect(len(r.ListOs())).To(Equal(0))
			Expect(len(r.ListK8s("x"))).To(Equal(0))
			Expect(r.GetInstaller("a","b")).To(BeNil())
                })
	        It("Should allow adding and gettnig installers", func() {
			r := NewRegistry()
			r.Add("ubuntu", "1.22", 122)
			r.Add("ubuntu", "1.23", 123)
			r.Add("rhel", "1.24", 124)

			Expect(r.GetInstaller("ubuntu","1.22")).To(Equal(122))
			Expect(r.GetInstaller("ubuntu","1.23")).To(Equal(123))
			Expect(r.GetInstaller("rhel","1.24")).To(Equal(124))
			Expect(len(r.ListOs())).To(Equal(2))
			Expect(len(r.ListK8s("ubuntu"))).To(Equal(2))
			Expect(len(r.ListK8s("rhel"))).To(Equal(1))
			Expect(r.GetInstaller("photon","1.22")).To(BeNil())
	        })
	})

})
