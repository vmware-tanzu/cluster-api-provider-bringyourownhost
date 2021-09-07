package hainstaller

import (
	"bytes"
	"fmt"
	"os/exec"
)

type StepSysCfg struct {
	Step
	ExecCmd []string
}

const defaultShell = "bash"

func (s StepSysCfg) Install() {
	println("swap off")

	for _, cmd := range s.ExecCmd {
		if len(cmd) > 0 {
			s.execCmd(cmd)
		}
	}

	println("firewall off")
}

func (s StepSysCfg) Uninstall() {
	println("swap on")

	for _, cmd := range s.ExecCmd {
		if len(cmd) > 0 {
			s.execCmd(cmd)
		}
	}

	println("firewall on")
}

/*
	Following is some extremely brutal shell exec code.
	Maybe we should discuss if it's okay to stay.

	However, I don't see any other way to run custom
	config commands at that point.
*/
func (s StepSysCfg) execCmd(cmd string) {
	stdOut, stdErr, err := s.shellExec(cmd)

	if len(stdOut) > 0 {
		fmt.Print(stdOut)
	}

	if len(stdErr) > 0 {
		fmt.Print(stdErr)
	}

	if err != nil {
		fmt.Print(err.Error())
	}
}

func (s StepSysCfg) shellExec(command string) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.Command(defaultShell, "-c", command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	return stdout.String(), stderr.String(), err
}
