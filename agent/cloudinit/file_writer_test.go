package cloudinit

import (
	"io/fs"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"strconv"
	"syscall"

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
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		err := os.RemoveAll(workDir)
		Expect(err).ToNot(HaveOccurred())
	})

	It("Should create a directory if it does not exists", func() {
		err := FileWriter{}.MkdirIfNotExists(workDir)
		Expect(err).ToNot(HaveOccurred())
	})

	It("Should not create a directory if it already exists", func() {
		FileWriter{}.MkdirIfNotExists(workDir)
		Expect(err).ToNot(HaveOccurred())

		err = FileWriter{}.MkdirIfNotExists(workDir)
		Expect(err).ToNot(HaveOccurred())
	})

	It("Should create and write to file", func() {
		fileName := path.Join(workDir, "file1.txt")
		fileContent := "some-content"

		err := FileWriter{}.MkdirIfNotExists(workDir)
		Expect(err).ToNot(HaveOccurred())

		err = FileWriter{}.WriteToFile(fileName, fileContent, "", "", false)
		Expect(err).NotTo(HaveOccurred())

		buffer, err := ioutil.ReadFile(fileName)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(buffer)).To(Equal(fileContent))

	})

	It("Should create and write to file with correct attributes", func() {

		fileName := path.Join(workDir, "file2.txt")
		userName := "root"
		groupName := "root"
		fileContent := "some-content"
		filePermission := 0777

		err := FileWriter{}.MkdirIfNotExists(workDir)
		Expect(err).ToNot(HaveOccurred())

		err = FileWriter{}.WriteToFile(fileName, fileContent, strconv.FormatInt(int64(filePermission), 8), userName+":"+groupName, false)
		Expect(err).NotTo(HaveOccurred())

		stats, err := os.Stat(fileName)
		Expect(err).NotTo(HaveOccurred())
		Expect(stats.Mode()).To(Equal(fs.FileMode(filePermission)))

		userInfo, err := user.Lookup(userName)
		Expect(err).NotTo(HaveOccurred())
		uid, err := strconv.ParseUint(userInfo.Uid, 10, 32)
		Expect(err).NotTo(HaveOccurred())
		gid, err := strconv.ParseUint(userInfo.Gid, 10, 32)
		Expect(err).NotTo(HaveOccurred())

		stat := stats.Sys().(*syscall.Stat_t)
		Expect(stat.Uid).To(Equal(uint32(uid)))
		Expect(stat.Gid).To(Equal(uint32(gid)))
	})

	It("Should append content to file when append mode is enabled", func() {
		fileName := path.Join(workDir, "file3.txt")
		fileOriginContent := "some-content-1"
		fileAppendContent := "some-content-2"

		err := FileWriter{}.MkdirIfNotExists(workDir)
		Expect(err).NotTo(HaveOccurred())

		err = ioutil.WriteFile(fileName, []byte(fileOriginContent), 0644)
		Expect(err).NotTo(HaveOccurred())

		err = FileWriter{}.WriteToFile(fileName, fileAppendContent, "", "", true)
		Expect(err).NotTo(HaveOccurred())

		buffer, err := ioutil.ReadFile(fileName)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(buffer)).To(Equal(fileOriginContent + fileAppendContent))

	})
})
