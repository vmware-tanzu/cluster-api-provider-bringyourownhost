package cloudinit

import (
	"io/ioutil"
	"os"
	"path"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("FileWriter", func() {
	var (
		dir string
		err error
	)

	BeforeEach(func() {
		dir, err = ioutil.TempDir("", "cloudinit")
		Expect(err).ToNot(HaveOccurred())
	})

	It("Should create a directory if it does not exists", func() {
		err := FileWriter{}.MkdirIfNotExists(path.Join(dir, "test"))

		Expect(err).ToNot(HaveOccurred())
	})

	It("Should not create a directory if it already exists", func() {
		FileWriter{}.MkdirIfNotExists(path.Join(dir, "test"))

		err = FileWriter{}.MkdirIfNotExists(path.Join(dir, "test"))

		Expect(err).ToNot(HaveOccurred())
	})

	It("Should create and write to file", func() {
		FileWriter{}.MkdirIfNotExists(path.Join(dir, "test"))

		err := FileWriter{}.WriteToFile(path.Join(dir, "test", "file1.txt"), "some-content")

		Expect(err).NotTo(HaveOccurred())
		buffer, err := ioutil.ReadFile(path.Join(dir, "test", "file1.txt"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(buffer)).To(Equal("some-content"))

	})

	AfterEach(func() {
		err := os.RemoveAll(dir)
		Expect(err).ToNot(HaveOccurred())
	})
})
