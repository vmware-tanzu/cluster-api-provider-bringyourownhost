package cloudinit

import (
	"fmt"
	"os/exec"
)

//counterfeiter:generate . ICmdRunner
type ICmdRunner interface {
	RunCmd(string) error
}

type CmdRunner struct {
}

func (r CmdRunner) RunCmd(cmd string) error {
	command := exec.Command("/bin/sh", "-c", cmd)
	output, err := command.Output()
	fmt.Println(string(output))
	return err
}
