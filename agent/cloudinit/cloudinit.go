package cloudinit

import (
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

type ScriptExecutor struct {
	WriteFilesExecutor IFileWriter
	RunCmdExecutor     ICmdRunner
}

type bootstrapConfig struct {
	FilesToWrite      []files  `json:"write_files"`
	CommandsToExecute []string `json:"runCmd"`
}

type files struct {
	Path string `json:"path,"`
	// Encoding    string `json:"encoding,omitempty"`
	// Owner       string `json:"owner,omitempty"`
	// Permissions string `json:"permissions,omitempty"`
	Content string `json:"content"`
	//Append      bool   `json:"append,"`
}

func (se ScriptExecutor) Execute(bootstrapScript string) error {
	cloudInitData := bootstrapConfig{}
	if err := yaml.Unmarshal([]byte(bootstrapScript), &cloudInitData); err != nil {
		return errors.Wrapf(err, "error parsing write_files action: %s", bootstrapScript)
	}

	for _, file := range cloudInitData.FilesToWrite {
		directoryToCreate := filepath.Dir(file.Path)
		err := se.WriteFilesExecutor.MkdirIfNotExists(directoryToCreate)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Error creating the directory %s", directoryToCreate))
		}

		err = se.WriteFilesExecutor.WriteToFile(file.Path, file.Content)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Error writing the file %s", file.Path))
		}
	}

	for _, cmd := range cloudInitData.CommandsToExecute {
		err := se.RunCmdExecutor.RunCmd(cmd)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Error running the command %s", cmd))
		}
	}
	return nil
}
