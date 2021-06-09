package cloudinit_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/cluster-api-provider-byoh/agent/cloudinit"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/agent/cloudinit/cloudinitfakes"
)

var _ = Describe("Cloudinit", func() {
	It("should execute the script", func() {
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
		// Eventually("/tmp/jme.txt").Should(BeAnExistingFile())

		// Eventually(func() string {
		// 	buffer, err := ioutil.ReadFile("/tmp/jme.txt")
		// 	if err != nil {
		// 		return ""
		// 	}
		// 	contents := string(buffer)
		// 	return contents
		// }).Should(Equal("is cooler than Anusha"))

	})

})
