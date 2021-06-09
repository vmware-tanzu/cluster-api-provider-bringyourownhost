package cloudinit_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestCloudinit(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cloudinit Suite")
}
