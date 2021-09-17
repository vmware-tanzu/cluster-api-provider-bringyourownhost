package installer

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
}

func (s *ShellStep) Do() {
	if len(s.DoCmd) > 0 {
		s.runStep(s.DoCmd)
	}
}

func (s *ShellStep) Undo() {
	if len(s.UndoCmd) > 0 {
		s.runStep(s.UndoCmd)
	}
}

func (s *ShellStep) runStep(command string) {
	var stdOut bytes.Buffer
	var stdErr bytes.Buffer

	const defaultShell = "bash"

	cmd := exec.Command(defaultShell, "-c", command)
	cmd.Stdout = &stdOut
	cmd.Stderr = &stdErr
	err := cmd.Run()

	if len(stdOut.String()) > 0 {
		fmt.Print(stdOut.String())
	}

	if len(stdErr.String()) > 0 {
		fmt.Print(stdErr)
	}

	if err != nil {
		fmt.Print(err.Error())
	}
}
