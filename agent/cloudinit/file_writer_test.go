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
		file := Files{
			Path:        path.Join(workDir, "file1.txt"),
			Encoding:    "",
			Owner:       "",
			Permissions: "",
			Content:     "some-content",
			Append:      false,
		}

		err := FileWriter{}.MkdirIfNotExists(workDir)
		Expect(err).ToNot(HaveOccurred())

		err = FileWriter{}.WriteToFile(file)
		Expect(err).NotTo(HaveOccurred())

		buffer, err := ioutil.ReadFile(file.Path)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(buffer)).To(Equal(file.Content))

	})

	It("Should create and write to file with correct attributes", func() {
		userName := "root"
		groupName := "root"
		filePermission := 0777
		file := Files{
			Path:        path.Join(workDir, "file2.txt"),
			Encoding:    "",
			Owner:       userName + ":" + groupName,
			Permissions: strconv.FormatInt(int64(filePermission), 8),
			Content:     "some-content",
			Append:      false,
		}

		err := FileWriter{}.MkdirIfNotExists(workDir)
		Expect(err).ToNot(HaveOccurred())

		err = FileWriter{}.WriteToFile(file)
		Expect(err).NotTo(HaveOccurred())

		stats, err := os.Stat(file.Path)
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
		//fileName := path.Join(workDir, "file3.txt")
		fileOriginContent := "some-content-1"
		//fileAppendContent := "some-content-2"

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

		err = FileWriter{}.WriteToFile(file)
		Expect(err).NotTo(HaveOccurred())

		buffer, err := ioutil.ReadFile(file.Path)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(buffer)).To(Equal(fileOriginContent + file.Content))

	})
})
