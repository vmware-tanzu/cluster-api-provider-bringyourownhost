// Copyright 2021 VMware, Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package cloudinit

import (
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("FileWriter", func() {

	var (
		workDir string
		err     error
	)

	BeforeEach(func() {
		workDir, err = ioutil.TempDir("", "file_writer_ut")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		err := os.RemoveAll(workDir)
		Expect(err).NotTo(HaveOccurred())
	})

	It("Should create a directory if it does not exists", func() {
		err := FileWriter{}.MkdirIfNotExists(workDir)
		Expect(err).NotTo(HaveOccurred())
	})

	It("Should not create a directory if it already exists", func() {
		err := FileWriter{}.MkdirIfNotExists(workDir)
		Expect(err).NotTo(HaveOccurred())

		err = FileWriter{}.MkdirIfNotExists(workDir)
		Expect(err).NotTo(HaveOccurred())
	})

	It("Should create and write to file", func() {
		filePermission := 0777
		file := Files{
			Path:        path.Join(workDir, "file1.txt"),
			Encoding:    "",
			Owner:       "",
			Permissions: strconv.FormatInt(int64(filePermission), 8),
			Content:     "some-content",
			Append:      false,
		}

		err := FileWriter{}.MkdirIfNotExists(workDir)
		Expect(err).NotTo(HaveOccurred())

		err = FileWriter{}.WriteToFile(&file)
		Expect(err).NotTo(HaveOccurred())

		buffer, err := ioutil.ReadFile(file.Path)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(buffer)).To(Equal(file.Content))

		stats, err := os.Stat(file.Path)
		Expect(err).NotTo(HaveOccurred())
		Expect(stats.Mode()).To(Equal(fs.FileMode(filePermission)))

	})

	It("Should append content to file when append mode is enabled", func() {
		fileOriginContent := "some-content-1"
		file := Files{
			Path:        path.Join(workDir, "file3.txt"),
			Encoding:    "",
			Owner:       "",
			Permissions: "",
			Content:     "some-content-2",
			Append:      true,
		}

		err := FileWriter{}.MkdirIfNotExists(workDir)
		Expect(err).NotTo(HaveOccurred())

		err = ioutil.WriteFile(file.Path, []byte(fileOriginContent), 0644)
		Expect(err).NotTo(HaveOccurred())

		err = FileWriter{}.WriteToFile(&file)
		Expect(err).NotTo(HaveOccurred())

		buffer, err := ioutil.ReadFile(file.Path)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(buffer)).To(Equal(fileOriginContent + file.Content))

	})
})
