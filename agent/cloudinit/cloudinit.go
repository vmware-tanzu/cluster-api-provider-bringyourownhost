package cloudinit

import (
	"path/filepath"

	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

type ScriptExecutor struct {
	Executor IFileWriter
}

type writeFilesAction struct {
	Files []files `json:"write_files,"`
}

type files struct {
	Path string `json:"path,"`
	// Encoding    string `json:"encoding,omitempty"`
	// Owner       string `json:"owner,omitempty"`
	// Permissions string `json:"permissions,omitempty"`
	Content string `json:"content,"`
	//Append      bool   `json:"append,"`
}

func (se ScriptExecutor) Execute(bootstrapScript string) error {
	cloudInitData := writeFilesAction{}
	if err := yaml.Unmarshal([]byte(bootstrapScript), &cloudInitData); err != nil {
		return errors.Wrapf(err, "error parsing write_files action: %s", bootstrapScript)
	}

	for _, file := range cloudInitData.Files {
		err := se.Executor.MkdirIfNotExists(filepath.Dir(file.Path), 0644)
		if err != nil {
			return err
		}

		err = se.Executor.WriteToFile(file.Path, file.Content)
		if err != nil {
			return err
		}

	}

	// defer f.Close()

	// if _, err := f.WriteString(cloudInitData.Files[0].Content); err != nil {
	// 	return errors.Wrapf(err, "failed to write file %s", path)
	// }

	return nil

}
