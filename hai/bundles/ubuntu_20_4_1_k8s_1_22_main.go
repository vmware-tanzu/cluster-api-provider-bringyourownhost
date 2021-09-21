package main

import (
	"os"

	"github.com/vmware-tanzu/cluster-api-provider-byoh/hai/bundles/installer/installer"
)

func main() {
	installer.RunInstaller(
		os.Args,
		&installer.BaseK8sInstaller{K8sStepProvider: &installer.Ubuntu_20_4_1_k8s_1_22{}})
}
