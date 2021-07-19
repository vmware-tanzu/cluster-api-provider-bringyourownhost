package cloudinit

import (
	"fmt"
	"os/exec"
	"strings"
)

//counterfeiter:generate . ICmdRunner
type ICmdRunner interface {
	RunCmd(string) error
}

type CmdRunner struct {
}

func (r CmdRunner) RunCmd(cmd string) error {
	subStrs := []string{"kubeadm init", "kubeadm join"}

	for _, subStr := range subStrs {
		if strings.Contains(cmd, subStr) {
			index := strings.Index(cmd, subStr)
			index += len(subStr)
			newCmd := cmd[:index] + " --ignore-preflight-errors=all " + cmd[index:]
			cmd = newCmd
		}
	}
	command := exec.Command("/bin/sh", "-c", cmd)
	output, err := command.Output()
	fmt.Println(string(output))
	return err
}
