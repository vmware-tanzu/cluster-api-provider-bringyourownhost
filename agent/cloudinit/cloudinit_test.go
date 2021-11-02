// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cloudinit_test

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/cloudinit"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/cloudinit/cloudinitfakes"
)

var _ = Describe("Cloudinit", func() {
	var (
		workDir string
		err     error
	)

	Context("Testing write_files and runCmd directives of cloudinit", func() {
		var (
			fakeFileWriter         *cloudinitfakes.FakeIFileWriter
			fakeCmdExecutor        *cloudinitfakes.FakeICmdRunner
			fakeTemplateParser     *cloudinitfakes.FakeITemplateParser
			scriptExecutor         cloudinit.ScriptExecutor
			defaultBootstrapSecret string
		)

		BeforeEach(func() {
			fakeFileWriter = &cloudinitfakes.FakeIFileWriter{}
			fakeCmdExecutor = &cloudinitfakes.FakeICmdRunner{}
			fakeTemplateParser = &cloudinitfakes.FakeITemplateParser{}
			scriptExecutor = cloudinit.ScriptExecutor{
				WriteFilesExecutor:    fakeFileWriter,
				RunCmdExecutor:        fakeCmdExecutor,
				ParseTemplateExecutor: fakeTemplateParser,
			}

			defaultBootstrapSecret = fmt.Sprintf(`write_files:
- path: %s/defaultFile.txt
  content: some-content
runCmd:
- echo 'some run command'`, workDir)

			workDir, err = ioutil.TempDir("", "cloudinit_ut")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			err = os.RemoveAll(workDir)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should write files successfully", func() {
			fileDir1 := path.Join(workDir, "dir1")
			fileName1 := path.Join(fileDir1, "file1.txt")
			fileContent1 := "some-unique-content-1"

			fileDir2 := path.Join(workDir, "dir2")
			fileName2 := path.Join(fileDir2, "file2.txt")
			fileContent2 := "some-uniqie-content-2"
			fileBase64Content := base64.StdEncoding.EncodeToString([]byte(fileContent2))
			permissions := "0777"
			encoding := "base64"

			bootstrapSecretUnencoded := fmt.Sprintf(`write_files:
- path: %s
  content: %s
- path: %s
  content: %s
  permissions: '%s'
  append: true
  encoding: %s`, fileName1, fileContent1, fileName2, fileBase64Content, permissions, encoding)

			err = scriptExecutor.Execute(bootstrapSecretUnencoded)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeFileWriter.MkdirIfNotExistsCallCount()).To(Equal(2))
			Expect(fakeFileWriter.WriteToFileCallCount()).To(Equal(2))
			Expect(fakeTemplateParser.ParseTemplateCallCount()).To(Equal(2))

			dirNameForFirstFile := fakeFileWriter.MkdirIfNotExistsArgsForCall(0)
			Expect(dirNameForFirstFile).To(Equal(fileDir1))
			firstFile := fakeFileWriter.WriteToFileArgsForCall(0)
			firstFile.Content = fakeTemplateParser.ParseTemplateArgsForCall(0)
			Expect(firstFile.Path).To(Equal(fileName1))
			Expect(firstFile.Content).To(Equal(fileContent1))

			dirNameForSecondFile := fakeFileWriter.MkdirIfNotExistsArgsForCall(1)
			Expect(dirNameForSecondFile).To(Equal(fileDir2))
			secondFile := fakeFileWriter.WriteToFileArgsForCall(1)
			secondFile.Content = fakeTemplateParser.ParseTemplateArgsForCall(1)
			Expect(secondFile.Path).To(Equal(fileName2))
			Expect(secondFile.Content).To(Equal(fileContent2))
			Expect(secondFile.Permissions).To(Equal(permissions))
			Expect(secondFile.Append).To(BeTrue())

		})

		It("should error out when an invalid yaml is passed", func() {
			err := scriptExecutor.Execute("invalid yaml")

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error parsing write_files action"))
		})

		It("should error out when there is not enough permission to mkdir", func() {
			fakeFileWriter.MkdirIfNotExistsReturns(errors.New("not enough permissions"))

			err := scriptExecutor.Execute(defaultBootstrapSecret)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not enough permissions"))
			Expect(fakeFileWriter.WriteToFileCallCount()).To(Equal(0))

		})

		It("should error out write to file failes", func() {
			fakeFileWriter.WriteToFileReturns(errors.New("cannot write to file"))

			err := scriptExecutor.Execute(defaultBootstrapSecret)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot write to file"))
		})

		It("run the command given in the runCmd directive", func() {
			err := scriptExecutor.Execute(defaultBootstrapSecret)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeCmdExecutor.RunCmdCallCount()).To(Equal(1))
			cmd := fakeCmdExecutor.RunCmdArgsForCall(0)
			Expect(cmd).To(Equal("echo 'some run command'"))
		})

		It("should not invoke the runCmd or writeFiles directive when absent", func() {

			err := scriptExecutor.Execute("")
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeCmdExecutor.RunCmdCallCount()).To(Equal(0))
			Expect(fakeFileWriter.MkdirIfNotExistsCallCount()).To(Equal(0))
			Expect(fakeFileWriter.WriteToFileCallCount()).To(Equal(0))
		})

		It("should error out when command execution fails", func() {
			fakeCmdExecutor.RunCmdReturns(errors.New("command execution failed"))
			err := scriptExecutor.Execute(defaultBootstrapSecret)
			Expect(err).To(HaveOccurred())

			Expect(fakeCmdExecutor.RunCmdCallCount()).To(Equal(1))

			Expect(err.Error()).To(ContainSubstring("command execution failed"))
		})
	})
})
