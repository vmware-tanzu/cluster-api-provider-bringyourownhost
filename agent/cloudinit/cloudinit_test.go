package cloudinit_test

import (
	"errors"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/cluster-api-provider-byoh/agent/cloudinit"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/agent/cloudinit/cloudinitfakes"
)

var _ = Describe("Cloudinit", func() {
	It("should write files successfully", func() {
		fakeFileWriter := &cloudinitfakes.FakeFileWriter{}

		se := cloudinit.ScriptExecutor{Executor: fakeFileWriter}
		bootstrapSecretUnencoded := `## template: jinja
#cloud-config
write_files:
-   path: /tmp/jme.txt
    content: is cooler than Anusha
`
		err := se.Execute(bootstrapSecretUnencoded)
		Expect(err).ToNot(HaveOccurred())

		Expect(fakeFileWriter.MkdirCallCount()).To(Equal(1))
		dirName, dirPermissions := fakeFileWriter.MkdirArgsForCall(0)
		Expect(dirName).To(Equal("/tmp"))
		Expect(dirPermissions).To(Equal(os.FileMode(0644)))

		Expect(fakeFileWriter.WriteToFileCallCount()).To(Equal(1))
		fileName, fileContents := fakeFileWriter.WriteToFileArgsForCall(0)
		Expect(fileName).To(Equal("/tmp/jme.txt"))
		Expect(fileContents).To(Equal("is cooler than Anusha"))

	})

	It("should error out when an invalid yaml is passed", func() {
		fakeFileWriter := &cloudinitfakes.FakeFileWriter{}

		se := cloudinit.ScriptExecutor{Executor: fakeFileWriter}
		bootstrapSecretUnencoded := "invalid yaml"
		err := se.Execute(bootstrapSecretUnencoded)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("error parsing write_files action"))

	})

	It("should error out when there is not enough permission to mkdir", func() {
		fakeFileWriter := &cloudinitfakes.FakeFileWriter{}

		se := cloudinit.ScriptExecutor{Executor: fakeFileWriter}
		bootstrapSecretUnencoded := `## template: jinja
#cloud-config
write_files:
-   path: /tmp/jme.txt
    content: is cooler than Anusha
`
		fakeFileWriter.MkdirReturns(errors.New("not enough permissions"))
		err := se.Execute(bootstrapSecretUnencoded)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("not enough permissions"))

		Expect(fakeFileWriter.WriteToFileCallCount()).To(Equal(0))

	})

	// It("should get all commands", func() {
	// 	bootstrapSecretUnencoded := `## template: jinja
	// 	#cloud-config
	// 	runCmd:
	// 	-   echo 'I am echo from the test'
	// 	`

	// })

})
