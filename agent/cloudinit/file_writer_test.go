// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cloudinit_test

import (
	"io/fs"
	"os"
	"path"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/cloudinit"
)

var _ = Describe("FileWriter", func() {

	var (
		workDir string
		err     error
	)

	BeforeEach(func() {
		workDir, err = os.MkdirTemp("", "file_writer_ut")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		err := os.RemoveAll(workDir)
		Expect(err).NotTo(HaveOccurred())
	})

	It("Should create a directory if it does not exists", func() {
		err := cloudinit.FileWriter{}.MkdirIfNotExists(workDir)
		Expect(err).NotTo(HaveOccurred())
	})

	It("Should not create a directory if it already exists", func() {
		err := cloudinit.FileWriter{}.MkdirIfNotExists(workDir)
		Expect(err).NotTo(HaveOccurred())

		err = cloudinit.FileWriter{}.MkdirIfNotExists(workDir)
		Expect(err).NotTo(HaveOccurred())
	})

	It("Should create and write to file", func() {
		filePermission := 0777
		file := cloudinit.Files{
			Path:        path.Join(workDir, "file1.txt"),
			Encoding:    "",
			Owner:       "",
			Permissions: strconv.FormatInt(int64(filePermission), 8),
			Content:     "some-content",
			Append:      false,
		}

		err := cloudinit.FileWriter{}.MkdirIfNotExists(workDir)
		Expect(err).NotTo(HaveOccurred())

		err = cloudinit.FileWriter{}.WriteToFile(&file)
		Expect(err).NotTo(HaveOccurred())

		buffer, err := os.ReadFile(file.Path)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(buffer)).To(Equal(file.Content))

		stats, err := os.Stat(file.Path)
		Expect(err).NotTo(HaveOccurred())
		Expect(stats.Mode()).To(Equal(fs.FileMode(filePermission)))

	})

	It("Should append content to file when append mode is enabled", func() {
		fileOriginContent := "some-file-content-1"
		file := cloudinit.Files{
			Path:        path.Join(workDir, "file3.txt"),
			Encoding:    "",
			Owner:       "",
			Permissions: "",
			Content:     "some-content-2",
			Append:      true,
		}

		err := cloudinit.FileWriter{}.MkdirIfNotExists(workDir)
		Expect(err).NotTo(HaveOccurred())

		err = os.WriteFile(file.Path, []byte(fileOriginContent), 0644)
		Expect(err).NotTo(HaveOccurred())

		err = cloudinit.FileWriter{}.WriteToFile(&file)
		Expect(err).NotTo(HaveOccurred())

		buffer, err := os.ReadFile(file.Path)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(buffer)).To(Equal(fileOriginContent + file.Content))

	})

	It("Should overwrite content of file when append mode is disabled", func() {
		fileOriginContent := "very long long message"
		file := cloudinit.Files{
			Path:        path.Join(workDir, "file3.txt"),
			Encoding:    "",
			Owner:       "",
			Permissions: "",
			Content:     "short message",
			Append:      false,
		}

		err := cloudinit.FileWriter{}.MkdirIfNotExists(workDir)
		Expect(err).NotTo(HaveOccurred())

		err = os.WriteFile(file.Path, []byte(fileOriginContent), 0644)
		Expect(err).NotTo(HaveOccurred())

		err = cloudinit.FileWriter{}.WriteToFile(&file)
		Expect(err).NotTo(HaveOccurred())

		buffer, err := os.ReadFile(file.Path)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(buffer)).To(Equal(file.Content))
	})

	It("should return error with invalid owner format", func() {
		file := cloudinit.Files{
			Path:        path.Join(workDir, "file1.txt"),
			Encoding:    "",
			Owner:       "root",
			Permissions: "",
			Content:     "some-content",
			Append:      false,
		}

		err := cloudinit.FileWriter{}.MkdirIfNotExists(workDir)
		Expect(err).NotTo(HaveOccurred())

		err = cloudinit.FileWriter{}.WriteToFile(&file)
		Expect(err).To(MatchError("invalid owner format 'root'"))
	})

	It("should return error with unknown owner", func() {
		file := cloudinit.Files{
			Path:        path.Join(workDir, "file1.txt"),
			Encoding:    "",
			Owner:       "some:random",
			Permissions: "",
			Content:     "some-content",
			Append:      false,
		}

		err := cloudinit.FileWriter{}.MkdirIfNotExists(workDir)
		Expect(err).NotTo(HaveOccurred())

		err = cloudinit.FileWriter{}.WriteToFile(&file)
		Expect(err).To(MatchError("Error Lookup user some: user: unknown user some"))
	})
})
