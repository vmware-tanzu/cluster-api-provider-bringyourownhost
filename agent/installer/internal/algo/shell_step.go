// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package algo

import (
	"bytes"
	"os/exec"
)

// ShellStep step that executes a shell command
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

	// TODO: check for exit(-1) or similar code
	cmd := exec.Command(defaultShell, "-c", command)
	s.OutputBuilder.Cmd(cmd.String())

	if s.BundlePath == "" {
		return nil
	}

	cmd.Stdout = &stdOut
	cmd.Stderr = &stdErr
	err := cmd.Run()

	if len(stdErr.String()) > 0 {
		/*
			this is a non-critical error
			the installer is still running properly
			but some package stderrored.
			e.g.:
				  - swap is already off;
				  - apt tells us it is installing from a local pkg
				    and cannot confirm the repository

			do not return err! just log it.
			otherwise, it will cause execution of the rollback procedure

			we only return error if the shellExec
			cannot be executed due to erroneous shell command, etc.
		*/
		s.OutputBuilder.Err(stdErr.String())
	}

	if err != nil {
		s.OutputBuilder.Err(err.Error())
		return err
	}

	if len(stdOut.String()) > 0 {
		s.OutputBuilder.Out(stdOut.String())
	}

	return nil
}
