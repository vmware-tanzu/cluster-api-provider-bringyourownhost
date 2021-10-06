package installer

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestInstaller(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Installer Suite")
}

var (
	archMap = map[string]string{
		"x64": "x86-64",
		"x32": "i686",
		"arm": "arm",
	}
)

func (osd *osDetector) mockHostSystemInfo(os, ver, arch string) (string, error) {
	out := "  Static hostname: ubuntu\n" +
		"        Icon name: computer-vm\n" +
		"          Chassis: vm\n" +
		"       Machine ID: 242642b0e734472abaf8c5337e1174c4\n" +
		"          Boot ID: 181f08d651b76h39be5b138231427c5c\n" +
		"   Virtualization: vmware\n" +
		" Operating System: " + os + " " + ver + " LTS\n" +
		"           Kernel: Linux 5.11.0-27-generic\n" +
		"     Architecture: " + archMap[arch] + "\n"

	return out, nil
}
