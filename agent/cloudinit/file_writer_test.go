package cloudinit

import (
	"io/ioutil"
	"os"
	"path"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/common"
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
		Expect(stats.Mode()).To(Equal(common.DefaultFileMode))

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
