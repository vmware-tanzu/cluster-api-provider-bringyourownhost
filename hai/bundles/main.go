package main

import (
	"os"

	"github.com/vmware-tanzu/cluster-api-provider-byoh/hai/bundles/installer"
)

func main() {

	if len(os.Args) < 1 {
		if os.Args[1] != "install" && os.Args[1] != "uninstall" {
			println("Please specify operation: install, uninstall")
		}

		os.Exit(-1)
	}

	builder := new(installer.Builder)
	bundle := builder.NewInstaller()

	if bundle == nil {
		println("Unsupported OS/TKG pair")
		os.Exit(-1)
	}

	switch os.Args[1] {
	case "install":
		println("\n- INSTALL -")
		bundle.Install()
	case "uninstall":
		println("\n- UNINSTALL -")
		bundle.Uninstall()
	}
}
