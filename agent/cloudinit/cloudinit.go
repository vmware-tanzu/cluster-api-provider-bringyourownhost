package cloudinit

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"path/filepath"
	"strings"

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
	Path        string `json:"path,"`
	Encoding    string `json:"encoding,omitempty"`
	Owner       string `json:"owner,omitempty"`
	Permissions string `json:"permissions,omitempty"`
	Content     string `json:"content"`
	Append      bool   `json:"append,omitempty"`
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

		encodings := fixEncoding(file.Encoding)
		content, err := fixContent(file.Content, encodings)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("error decoding content for %s", file.Path))
		}

		err = se.WriteFilesExecutor.WriteToFile(file.Path, content, file.Permissions, file.Owner, file.Append)
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

func fixEncoding(e string) []string {
	e = strings.ToLower(e)
	e = strings.TrimSpace(e)

	if e == "gz+base64" || e == "gzip+base64" || e == "gz+b64" || e == "gzip+b64" {
		return []string{"application/base64", "application/x-gzip"}
	} else if e == "base64" || e == "b64" {
		return []string{"application/base64"}
	}

	return []string{"text/plain"}
}

func fixContent(content string, encodings []string) (string, error) {
	for _, e := range encodings {
		switch e {
		case "application/base64":
			rByte, err := base64.StdEncoding.DecodeString(content)
			if err != nil {
				return content, errors.WithStack(err)
			}
			content = string(rByte)
		case "application/x-gzip":
			rByte, err := gUnzipData([]byte(content))
			if err != nil {
				return content, err
			}
			content = string(rByte)
		case "text/plain":
			continue
		default:
			return content, errors.Errorf("Unknown bootstrap data encoding: %q", content)
		}
	}
	return content, nil
}

func gUnzipData(data []byte) ([]byte, error) {
	var r io.Reader
	var err error
	b := bytes.NewBuffer(data)
	r, err = gzip.NewReader(b)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var resB bytes.Buffer
	_, err = resB.ReadFrom(r)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return resB.Bytes(), nil
}
