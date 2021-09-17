package main

import (
	"os"

	"github.com/vmware-tanzu/cluster-api-provider-byoh/hai/bundles/installer"
)

func main() {

	if len(os.Args) < 1 {
		os.Exit(-1)
	}

	installer := &installer.BaseK8sInstaller{
		K8sInstaller: &installer.Ubuntu_20_4_1_tkg_1_22{}}

	switch os.Args[1] {
	case "install":
		installer.Install()
	case "uninstall":
		installer.Uninstall()
	default:
		println("Please specify operation: install, uninstall")
	}
}
