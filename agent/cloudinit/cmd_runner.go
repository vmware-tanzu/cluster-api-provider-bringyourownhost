// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cloudinit

import (
	"context"
	"os"
	"os/exec"
)

//counterfeiter:generate . ICmdRunner
type ICmdRunner interface {
	RunCmd(context.Context, string) error
}

// CmdRunner default implementer of ICmdRunner
// TODO reevaluate empty interface/struct
type CmdRunner struct {
}

// RunCmd executes the command string
func (r CmdRunner) RunCmd(ctx context.Context, cmd string) error {
	command := exec.CommandContext(ctx, "/bin/bash", "-c", cmd)
	command.Stderr = os.Stderr
	command.Stdout = os.Stdout
	if err := command.Run(); err != nil {
		return err
	}
	return nil
}
