/*
Copyright the Cluster API Provider BYOH contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cloudinit

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

// writeFilesAction defines a list of files that should be written to a node.
type writeFilesAction struct {
	Files []files `json:"write_files,"`
}

type files struct {
	Path        string `json:"path,"`
	Encoding    string `json:"encoding,omitempty"`
	Owner       string `json:"owner,omitempty"`
	Permissions string `json:"permissions,omitempty"`
	Content     string `json:"content,"`
	Append      bool   `json:"append,"`
}

func newWriteFilesAction() action {
	return &writeFilesAction{}
}

func (a *writeFilesAction) Unmarshal(userData []byte) error {
	if err := yaml.Unmarshal(userData, a); err != nil {
		return errors.Wrapf(err, "error parsing write_files action: %s", userData)
	}
	return nil
}

func (a *writeFilesAction) Run(ctx context.Context, logger logr.Logger) error {

	for _, f := range a.Files {
		// Fix attributes and apply defaults
		path := fixPath(f.Path)
		encodings := fixEncoding(f.Encoding)
		content, err := fixContent(f.Content, encodings)
		if err != nil {
			return errors.Wrapf(err, "error decoding content for %s", path)
		}

		logger.Info("cloud init adapter:: write_files", "path", path)

		// Create the target directory if missing
		directory := filepath.Dir(path)
		if _, err := os.Stat(directory); os.IsNotExist(err) {
			if err := os.Mkdir(directory, 0644); err != nil {
				return errors.Wrapf(err, "failed to create folder %s", directory)
			}
		}

		// Create the file
		// TODO: implement support for f.Append
		f, err := os.Create(path)
		if err != nil {
			return errors.Wrapf(err, "failed to create file %s", path)
		}
		defer f.Close()

		if _, err := f.WriteString(content); err != nil {
			return errors.Wrapf(err, "failed to write file %s", path)
		}

		// TODO: implement support for setting file permissions
		// TODO: implement supprot for setting file ownership
	}
	return nil
}

func fixPath(p string) string {
	return strings.TrimSpace(p)
}

func fixEncoding(e string) []string {
	e = strings.ToLower(e)
	e = strings.TrimSpace(e)

	switch e {
	case "gz", "gzip":
		return []string{"application/x-gzip"}
	case "gz+base64", "gzip+base64", "gz+b64", "gzip+b64":
		return []string{"application/base64", "application/x-gzip"}
	case "base64", "b64":
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
			return string(rByte), nil
		case "application/x-gzip":
			rByte, err := gUnzipData([]byte(content))
			if err != nil {
				return content, err
			}
			return string(rByte), nil
		case "text/plain":
			return content, nil
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
