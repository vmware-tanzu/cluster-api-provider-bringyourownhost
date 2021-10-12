package installer

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Byohost Installer Tests", func() {
	 Context("When installer is created", func() {
                It("Should return error", func() {
			i, err := New("repo", "downloadPath", nil)
			Expect(err).Should((HaveOccurred()))
                })
	})

})
