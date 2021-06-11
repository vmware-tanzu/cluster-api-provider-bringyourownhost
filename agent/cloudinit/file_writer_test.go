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
		err := FileWriter{}.MkdirIfNotExists(path.Join(dir, "test"), 0644)

		Expect(err).ToNot(HaveOccurred())
	})

	It("Should not create a directory if it already exists", func() {
		FileWriter{}.MkdirIfNotExists(path.Join(dir, "test"), 0644)

		err = FileWriter{}.MkdirIfNotExists(path.Join(dir, "test"), 0644)

		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		err := os.RemoveAll(dir)
		Expect(err).ToNot(HaveOccurred())
	})
})
