package cloudinit_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/cluster-api-provider-byoh/agent/cloudinit"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/agent/cloudinit/cloudinitfakes"
)

var someBootstrapSecret = `
write_files:
-   path: /tmp/file1.txt
    content: some-content
runCmd:
-   echo 'some run command'
`

var _ = Describe("Cloudinit", func() {
	Context("Testing write_files and runCmd directives of cloudinit", func() {
		var (
			fakeFileWriter  *cloudinitfakes.FakeIFileWriter
			fakeCmdExecutor *cloudinitfakes.FakeICmdRunner
			scriptExecutor  cloudinit.ScriptExecutor
			err             error
		)

		BeforeEach(func() {
			fakeFileWriter = &cloudinitfakes.FakeIFileWriter{}
			fakeCmdExecutor = &cloudinitfakes.FakeICmdRunner{}
			scriptExecutor = cloudinit.ScriptExecutor{
				WriteFilesExecutor: fakeFileWriter,
				RunCmdExecutor:     fakeCmdExecutor,
			}
		})

		It("should write files successfully", func() {
			bootstrapSecretUnencoded := `write_files:
-   path: /tmp/a/file1.txt
    content: some-content
-   path: /tmp/b/file2.txt
    content: whatever`
			err = scriptExecutor.Execute(bootstrapSecretUnencoded)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeFileWriter.MkdirIfNotExistsCallCount()).To(Equal(2))
			Expect(fakeFileWriter.WriteToFileCallCount()).To(Equal(2))

			dirNameForFirstFile := fakeFileWriter.MkdirIfNotExistsArgsForCall(0)
			Expect(dirNameForFirstFile).To(Equal("/tmp/a"))
			firstFileName, firstFileContents := fakeFileWriter.WriteToFileArgsForCall(0)
			Expect(firstFileName).To(Equal("/tmp/a/file1.txt"))
			Expect(firstFileContents).To(Equal("some-content"))

			dirNameForSecondFile := fakeFileWriter.MkdirIfNotExistsArgsForCall(1)
			Expect(dirNameForSecondFile).To(Equal("/tmp/b"))
			secondFileName, secondFileContents := fakeFileWriter.WriteToFileArgsForCall(1)
			Expect(secondFileName).To(Equal("/tmp/b/file2.txt"))
			Expect(secondFileContents).To(Equal("whatever"))
		})

		It("should error out when an invalid yaml is passed", func() {
			err := scriptExecutor.Execute("invalid yaml")

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error parsing write_files action"))
		})

		It("should error out when there is not enough permission to mkdir", func() {
			fakeFileWriter.MkdirIfNotExistsReturns(errors.New("not enough permissions"))

			err := scriptExecutor.Execute(someBootstrapSecret)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not enough permissions"))
			Expect(fakeFileWriter.WriteToFileCallCount()).To(Equal(0))

		})

		It("should error out write to file failes", func() {
			fakeFileWriter.WriteToFileReturns(errors.New("cannot write to file"))

			err := scriptExecutor.Execute(someBootstrapSecret)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot write to file"))
		})

		It("run the command given in the runCmd directive", func() {
			err := scriptExecutor.Execute(someBootstrapSecret)
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
			err := scriptExecutor.Execute(someBootstrapSecret)
			Expect(err).To(HaveOccurred())

			Expect(fakeCmdExecutor.RunCmdCallCount()).To(Equal(1))

			Expect(err.Error()).To(ContainSubstring("command execution failed"))
		})
	})
})
