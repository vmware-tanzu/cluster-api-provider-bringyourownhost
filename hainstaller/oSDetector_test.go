// Tests for the oSDetector class

package hainstaller

import (
	"strings"
	"testing"
)

func TestFilterSystemInfo(t *testing.T) {
	d := newOSDetector()
	systemInfo :=
		"  Static hostname: ubuntu\n" +
			"        Icon name: computer-vm\n" +
			"          Chassis: vm\n" +
			"       Machine ID: 242642b0e734472abaf8c5337e1174c4\n" +
			"          Boot ID: 181f08d651b76h39be5b138231427c5c\n" +
			"   Virtualization: vmware\n" +
			" Operating System: Ubuntu 20.04.3 LTS\n" +
			"           Kernel: Linux 5.11.0-27-generic\n" +
			"     Architecture: x86-64\n"
	os, ver, arch := d.filterSystemInfo(systemInfo)
	os = strings.TrimSpace(os)
	expectedOS := "Ubuntu"
	if os != expectedOS {
		t.Errorf("OS name was incorrect, got: %s, want: %s", os, expectedOS)
	}
	expectedVer := "20.04.3"
	if ver != expectedVer {
		t.Errorf("Verion was incorrect, got: %s, want: %s", ver, expectedVer)
	}
	expectedArch := "x86-64"
	if arch != expectedArch {
		t.Errorf("Architecture was incorrect, got: %s, want: %s", arch, expectedArch)
	}
}

func TestNormalizeOsName(t *testing.T) {
	d := newOSDetector()
	os := "Ubuntu"
	ver := "20.04.3"
	arch := "x86-64"
	normOS := d.normalizeOsName(os, ver, arch)
	expectedNormOS := "Ubuntu_20.04.3_x64"
	if normOS != expectedNormOS {
		t.Errorf("Normalized OS name was incorrect, got: %s, want: %s", normOS, expectedNormOS)
	}
}
