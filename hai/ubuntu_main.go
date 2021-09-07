/*
	This is an example for installing .deb packages on Ubuntu
	It's a work in progress.
	It extends the main Bundle struct of hainstaller.go
*/

package main

import (
	"hai/hainstaller"
)

type Ubuntu20_04 struct {
	hainstaller.Bundle
}

var ubt Ubuntu20_04

//extend the default Bundle.Install()
func init() {
	println("Ubuntu 20.04 Extension")

	extraUbtStep := new(hainstaller.StepSysCfg)
	extraUbtStep.ExecCmd = append(extraUbtStep.ExecCmd, "ls -a")
	ubt.AddStep(extraUbtStep)

	step1 := new(hainstaller.StepBackup)
	step1.Item = "package1.deb"
	step2 := new(hainstaller.StepBackup)
	step2.Item = "package2.deb"
	stepCfg := new(hainstaller.StepSysCfg)
	stepCfg.ExecCmd = append(stepCfg.ExecCmd, "ls -lh")

	ubt.AddStep(step1)
	ubt.AddStep(step2)
	ubt.AddStep(stepCfg)
}

func main() {
	ubt.Install("/path/to/ubuntu/image")
	println("")
	ubt.Uninstall("/path/to/ubuntu/image")
}
