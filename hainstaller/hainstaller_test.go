// Currently used for local testing purposes

package hainstaller

import (
	"testing"
)

// Test for Ubuntu 20.04 64 bit
func TestGetBundleName(t *testing.T) {

	hai := NewHostAgentInstaller("placeholder", "placeholder")
	bundleName, _ := hai.getBundleName("1.2.1")
	expected := "Ubuntu_20.04_x64_k8s_1.2.1"
	if bundleName != expected {
		t.Errorf("Bundle name was incorrect, got: %s, want: %s", bundleName, expected)
	}
}
