package algo

import (
	"bytes"
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
	*BaseK8sInstaller
}

func (s *ShellStep) do() error {
	s.OutputBuilder.Msg("Installing: " + s.Desc)
	return s.runStep(s.DoCmd)
}

func (s *ShellStep) undo() error {
	s.OutputBuilder.Msg("Uninstalling: " + s.Desc)
	return s.runStep(s.UndoCmd)
}

func (s *ShellStep) runStep(command string) error {
	var stdOut bytes.Buffer
	var stdErr bytes.Buffer

	const defaultShell = "bash"

	//TODO: check for exit(-1) or similar code
	cmd := exec.Command(defaultShell, "-c", command)
	s.OutputBuilder.Cmd(cmd.String())

	cmd.Stdout = &stdOut
	cmd.Stderr = &stdErr
	err := cmd.Run()

	if len(stdErr.String()) > 0 {
		//this is a non critical error (warning)
		//do not return err! just log it.
		//otherwise it will cause rollback procedure
		s.OutputBuilder.StdErr(stdErr.String())
	}

	if err != nil {
		s.OutputBuilder.StdErr(err.Error())
		return err
	}

	if len(stdOut.String()) > 0 {
		s.OutputBuilder.StdOut(stdOut.String())
	}

	return nil
}
