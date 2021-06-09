package cloudinit

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

type ScriptExecutor struct {
	Executor FileWriter
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

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate . FileWriter
type FileWriter interface {
	Mkdir(string, os.FileMode)
	// CreateFile(string)
	WriteToFile(string, string)
}

type RealWhatever struct {
}

func (w RealWhatever) Mkdir(dirName string, fileMode os.FileMode) {
	os.Mkdir(dirName, fileMode)
}

// func (w RealWhatever) CreateFile(fileName string) {
// 	os.Create(fileName)
// }

func (w RealWhatever) WriteToFile(fileName string, fileContent string) {
	f, _ := os.Create(fileName)
	defer f.Close()
	f.WriteString(fileContent)
}

func (se ScriptExecutor) Execute(bootstrapScript string) error {
	cloudInitData := writeFilesAction{}
	if err := yaml.Unmarshal([]byte(bootstrapScript), &cloudInitData); err != nil {
		return errors.Wrapf(err, "error parsing write_files action: %s", bootstrapScript)
	}

	path := cloudInitData.Files[0].Path
	directory := filepath.Dir(path)
	// os.Mkdir(directory, 0644)
	se.Executor.Mkdir(directory, 0644)
	// f, _ := os.Create(path)
	se.Executor.WriteToFile(path, cloudInitData.Files[0].Content)

	// defer f.Close()

	// if _, err := f.WriteString(cloudInitData.Files[0].Content); err != nil {
	// 	return errors.Wrapf(err, "failed to write file %s", path)
	// }

	return nil

}
