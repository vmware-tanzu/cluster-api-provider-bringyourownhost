package installer

import (
	"bytes"
	"fmt"
	"os/exec"
)

type step struct {
	cmd  string
	undo string
}

func (s *step) Execute() {
	if len(s.cmd) > 0 {
		s.runStep(s.cmd)
	}
}

func (s *step) Undo() {
	if len(s.undo) > 0 {
		s.runStep(s.undo)
	}
}

func (s *step) runStep(command string) {
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
