package cloudinit

import (
	"os"
	"os/exec"
)

//counterfeiter:generate . ICmdRunner
type ICmdRunner interface {
	RunCmd(string) error
}

type CmdRunner struct {
}

// RunCmd executes the command string
func (r CmdRunner) RunCmd(cmd string) error {
	command := exec.Command("/bin/sh", "-c", cmd)
	command.Stderr = os.Stderr
	return command.Run()
}
