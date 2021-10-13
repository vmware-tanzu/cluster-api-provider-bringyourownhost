package installer

import (
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Byohost Installer Tests", func() {
	Context("When installer is created", func() {
		It("Should return error", func() {
			_, err := New("repo", "downloadPath", logr.Logger{})
			Expect(err).ShouldNot((HaveOccurred()))
		})
	})

})
