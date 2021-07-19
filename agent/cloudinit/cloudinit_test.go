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
	"github.com/vmware-tanzu/cluster-api-provider-byoh/agent/cloudinit"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/agent/cloudinit/cloudinitfakes"
)

var _ = Describe("Cloudinit", func() {
	var (
		workDir string
		err     error
	)

	BeforeEach(func() {
		workDir, err = ioutil.TempDir("", "cloudinit_ut")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		err := os.RemoveAll(workDir)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("Testing write_files and runCmd directives of cloudinit", func() {
		var (
			fakeFileWriter         *cloudinitfakes.FakeIFileWriter
			fakeCmdExecutor        *cloudinitfakes.FakeICmdRunner
			scriptExecutor         cloudinit.ScriptExecutor
			defaultBootstrapSecret string
		)

		BeforeEach(func() {
			fakeFileWriter = &cloudinitfakes.FakeIFileWriter{}
			fakeCmdExecutor = &cloudinitfakes.FakeICmdRunner{}
			scriptExecutor = cloudinit.ScriptExecutor{
				WriteFilesExecutor: fakeFileWriter,
				RunCmdExecutor:     fakeCmdExecutor,
			}

			defaultBootstrapSecret = fmt.Sprintf(`write_files:
- path: %s/defaultFile.txt
  content: some-content
runCmd:
- echo 'some run command'`, workDir)
		})

		AfterEach(func() {
			err := os.RemoveAll(workDir)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should write files successfully", func() {
			fileDir1 := path.Join(workDir, "dir1")
			fileName1 := path.Join(fileDir1, "file1.txt")
			fileContent1 := "some-content-1"

			fileDir2 := path.Join(workDir, "dir2")
			fileName2 := path.Join(fileDir2, "file2.txt")
			fileContent2 := "some-content-2"

			bootstrapSecretUnencoded := fmt.Sprintf(`write_files:
- path: %s
  content: %s
- path: %s
  content: %s`, fileName1, fileContent1, fileName2, fileContent2)

			err = scriptExecutor.Execute(bootstrapSecretUnencoded)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeFileWriter.MkdirIfNotExistsCallCount()).To(Equal(2))
			Expect(fakeFileWriter.WriteToFileCallCount()).To(Equal(2))

			dirNameForFirstFile := fakeFileWriter.MkdirIfNotExistsArgsForCall(0)
			Expect(dirNameForFirstFile).To(Equal(fileDir1))
			firstFileName, firstFileContents := fakeFileWriter.WriteToFileArgsForCall(0)
			Expect(firstFileName).To(Equal(fileName1))
			Expect(firstFileContents).To(Equal(fileContent1))

			dirNameForSecondFile := fakeFileWriter.MkdirIfNotExistsArgsForCall(1)
			Expect(dirNameForSecondFile).To(Equal(fileDir2))
			secondFileName, secondFileContents := fakeFileWriter.WriteToFileArgsForCall(1)
			Expect(secondFileName).To(Equal(fileName2))
			Expect(secondFileContents).To(Equal(fileContent2))
		})

		It("could recognize owner, permissions, and append attributes", func() {

			fileDir := path.Join(workDir, "dir")
			fileName := path.Join(fileDir, "file.txt")
			fileContent := "some-content-append"
			fileBase64Content := base64.StdEncoding.EncodeToString([]byte(fileContent))
			user := "root"
			group := "root"
			permissions := "0777"
			encoding := "base64"

			bootstrapSecretUnencoded := fmt.Sprintf(`write_files:
- path: %s
  content: %s
  owner: %s:%s
  permissions: '%s'
  append: true
  encoding: %s`, fileName, fileBase64Content, user, group, permissions, encoding)

			err = scriptExecutor.Execute(bootstrapSecretUnencoded)
			Expect(err).ToNot(HaveOccurred())
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
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeCmdExecutor.RunCmdCallCount()).To(Equal(1))
			cmd := fakeCmdExecutor.RunCmdArgsForCall(0)
			Expect(cmd).To(Equal("echo 'some run command'"))
		})

		It("should not invoke the runCmd or writeFiles directive when absent", func() {

			err := scriptExecutor.Execute("")
			Expect(err).ToNot(HaveOccurred())

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
