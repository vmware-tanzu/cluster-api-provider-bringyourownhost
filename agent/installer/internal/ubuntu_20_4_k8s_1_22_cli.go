package main

import (
	"os"

	"github.com/vmware-tanzu/cluster-api-provider-byoh/agent/installer/internal/algo"
)

func main() {
	algo.RunInstaller(
		os.Args[1], //operation
		os.Args[2], //bundlePath
		algo.BaseK8sInstaller{
			K8sStepProvider: &algo.Ubuntu_20_4_k8s_1_22{}})

}
