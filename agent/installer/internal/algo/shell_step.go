package algo

import (
	"bytes"
	"fmt"
	"os/exec"
)

/*
########################################
# Extends Step to a shell exec command #
########################################
*/
type ShellStep struct {
	Step
	DoCmd   string
	UndoCmd string
	Desc    string
}

const (
	PIPE_ERR  = 0
	PIPE_OUT  = 1
	PIPE_INFO = 2
)

func (s *ShellStep) do(k *BaseK8sInstaller) error {
	infoStr := "Processing install step: " + s.Desc
	infoStr += "\nEXECUTING: " + s.DoCmd + "\n"
	s.logStep(k, PIPE_INFO, infoStr)

	return s.runStep(s.DoCmd, k)
}

func (s *ShellStep) undo(k *BaseK8sInstaller) error {
	infoStr := "Processing uninstall step: " + s.Desc
	infoStr += "\nEXECUTING: " + s.UndoCmd + "\n"
	s.logStep(k, PIPE_INFO, infoStr)

	return s.runStep(s.UndoCmd, k)
}

func (s *ShellStep) runStep(command string, k *BaseK8sInstaller) error {
	var stdOut bytes.Buffer
	var stdErr bytes.Buffer

	const defaultShell = "bash"

	//TODO: check for exit(-1) or similar code
	cmd := exec.Command(defaultShell, "-c", command)
	cmd.Stdout = &stdOut
	cmd.Stderr = &stdErr
	err := cmd.Run()

	if len(stdOut.String()) > 0 {
		s.logStep(k, PIPE_OUT, stdOut.String())
	}

	if len(stdErr.String()) > 0 {
		s.logStep(k, PIPE_ERR, stdErr.String())
	}

	if err != nil {
		fmt.Print(err.Error())
		s.logStep(k, PIPE_ERR, err.Error())
	}

	return nil
}

func (s *ShellStep) logStep(k *BaseK8sInstaller, outputPipe int, msg string) {
	switch outputPipe {
	case PIPE_ERR:
		k.LogBuilder.AddTimestamp(&k.LogBuilder.stdErr).AddStdErr(msg)
	case PIPE_OUT:
		k.LogBuilder.AddTimestamp(&k.LogBuilder.stdOut).AddStdOut(msg)
	case PIPE_INFO:
		k.LogBuilder.AddTimestamp(&k.LogBuilder.info).AddInfoText(msg)
	default:
	}

	println(k.LogBuilder.GetLastInfo())
	println(k.LogBuilder.GetLastStdOut())
	println(k.LogBuilder.GetLastStdErr())
}
