package cloudinit

import "os/exec"

//counterfeiter:generate . ICmdRunner
type ICmdRunner interface {
	RunCmd(string) error
}

type CmdRunner struct {
}

func (r CmdRunner) RunCmd(cmd string) error {

	command := exec.Command("/bin/sh", "-c", cmd)
	return command.Run()

}
