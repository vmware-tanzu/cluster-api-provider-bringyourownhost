package main

import (
	"os"

	"github.com/vmware-tanzu/cluster-api-provider-byoh/hai/bundles/installer/installer"
)

func main() {
	installer.RunInstaller(
		os.Args, new(installer.Ubuntu_20_4_tkg_1_22))
}
