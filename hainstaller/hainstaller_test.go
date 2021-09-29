// Currently used for local testing purposes

package hainstaller

import (
	"testing"
)

// Test for Ubuntu 20.04.3 64 bit
func TestOSDetectorDetect(t *testing.T) {

	d := newOSDetector()
	os, err := d.detect()
	if err != nil {
		t.Errorf("Could not get system info.")
	}
	expected := "Ubuntu_20.04.3_x64"
	if os != expected {
		t.Errorf("Bundle name was incorrect, got: %s, want: %s", os, expected)
	}
}

// Test for hai
// Test only for filterSystemInfo and normalizeOsName
// Test for install/uninstall bundle
// Test for registry
// Test for BundleDownloader
