package cloudinit

import (
	"os"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate . IFileWriter
type IFileWriter interface {
	MkdirIfNotExists(string, os.FileMode) error
	WriteToFile(string, string) error
}

type FileWriter struct {
}

func (w FileWriter) MkdirIfNotExists(dirName string, fileMode os.FileMode) error {
	_, err := os.Stat(dirName)

	if os.IsNotExist(err) {
		return os.Mkdir(dirName, fileMode)
	}
	return nil

}

func (w FileWriter) WriteToFile(fileName string, fileContent string) error {
	f, _ := os.Create(fileName)
	defer f.Close()
	_, err := f.WriteString(fileContent)
	return err
}
