package main

import (
	"log"
	"os"

	"github.com/vmware-tanzu/cluster-api-provider-byoh/agent/installer/internal/algo"
)

type ConsoleLogPrinter struct {
	algo.OutputBuilder
}

func (*ConsoleLogPrinter) StdOut(str string) {
	log.Print(str)
}

func (*ConsoleLogPrinter) StdErr(str string) {
	log.Print(str)
}

func (*ConsoleLogPrinter) Cmd(str string) {
	log.Print(str)
}

func (*ConsoleLogPrinter) Desc(str string) {
	log.Print(str)
}

func (*ConsoleLogPrinter) Msg(str string) {
	log.Print(str)
}

func main() {

	ubuntu := algo.Ubuntu_20_4_k8s_1_22{}
	ubuntu.OutputBuilder = &ConsoleLogPrinter{}
	ubuntu.BundlePath = os.Args[2]

	algo.RunInstaller(
		os.Args[1], //operation
		algo.BaseK8sInstaller{
			K8sStepProvider: &ubuntu,
			OutputBuilder:   ubuntu.OutputBuilder})
}
