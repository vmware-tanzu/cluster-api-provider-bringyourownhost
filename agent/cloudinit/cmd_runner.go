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

	if strings.Contains(cmd, "kubeadm") {
		cmd = "kubeadm join --config /run/kubeadm/kubeadm-join-config.yaml  --ignore-preflight-errors=all && echo success > /run/cluster-api/bootstrap-success.complete"
	}
	command := exec.Command("/bin/sh", "-c", cmd)
	output, err := command.Output()
	fmt.Println(string(output))
	return err

}
