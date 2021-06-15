package cloudinit

import (
	"os"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate . IFileWriter
type IFileWriter interface {
	MkdirIfNotExists(string) error
	WriteToFile(string, string) error
}

type FileWriter struct {
}

func (w FileWriter) MkdirIfNotExists(dirName string) error {
	_, err := os.Stat(dirName)

	if os.IsNotExist(err) {
		return os.MkdirAll(dirName, 0744)
	}

	if err != nil {
		return err
	}
	return nil

}

func (w FileWriter) WriteToFile(fileName string, fileContent string) error {
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}

	defer f.Close()
	_, err = f.WriteString(fileContent)
	return err
}
