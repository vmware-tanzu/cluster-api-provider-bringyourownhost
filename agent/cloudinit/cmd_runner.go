// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cloudinit

import (
	"os"
	"os/exec"
)

//counterfeiter:generate . ICmdRunner
type ICmdRunner interface {
	RunCmd(string) error
}

// CmdRunner default implementer of ICmdRunner
// TODO reevaluate empty interface/struct
type CmdRunner struct {
}

// RunCmd executes the command string
func (r CmdRunner) RunCmd(cmd string) error {
	command := exec.Command("/bin/sh", "-c", cmd)
	command.Stderr = os.Stderr
	return command.Run()
}
