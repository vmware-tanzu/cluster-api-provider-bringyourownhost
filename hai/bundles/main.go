package main

import (
	"os"

	"github.com/vmware-tanzu/cluster-api-provider-byoh/hai/bundles/installer"
)

func main() {

	if len(os.Args) == 2 && len(os.Args[1]) > 0 {
		installBundle(os.Args[1])
	} else if len(os.Args) == 3 && os.Args[1] == "--print" && len(os.Args[2]) > 0 {
		printInstallSteps(os.Args[2])
	} else {
		println("Please specify an OS\nExample: ubuntu_20_04_1")
	}
}

func installBundle(os string) {
	var builder installer.Builder
	bundle := builder.NewInstaller(os)

	if bundle != nil {
		bundle.Init()
		println("- INSTALL -")
		bundle.Install()
		println("- UNINSTALL -")
		bundle.Uninstall()
	} else {
		println("Unsupported or no OS has been specified")
	}
}

func printInstallSteps(os string) {
	var builder installer.Builder
	bundle := builder.NewInstaller(os)
	bundle.Init()
	bundle.PrintSteps()
}
