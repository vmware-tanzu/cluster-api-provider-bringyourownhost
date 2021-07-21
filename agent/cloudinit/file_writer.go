package cloudinit

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"

	"github.com/pkg/errors"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate . IFileWriter
type IFileWriter interface {
	MkdirIfNotExists(string) error
	WriteToFile(Files) error
}

type FileWriter struct {
}

func (w FileWriter) MkdirIfNotExists(dirName string) error {
	_, err := os.Stat(dirName)

	if os.IsNotExist(err) {
		return os.MkdirAll(dirName, 0744)
	}

	if err != nil {
		return err
	}
	return nil

}

func (w FileWriter) WriteToFile(file Files) error {
	initPermission := fs.FileMode(0644)
	if stats, err := os.Stat(file.Path); os.IsExist(err) {
		initPermission = stats.Mode()
	}

	flag := os.O_WRONLY | os.O_CREATE
	if file.Append {
		flag |= os.O_APPEND
	}

	f, err := os.OpenFile(file.Path, flag, initPermission)
	if err != nil {
		return err
	}

	defer f.Close()
	_, err = f.WriteString(file.Content)
	if err != nil {
		return err
	}

	u, err := user.Current()
	if err != nil {
		return err
	}
	if len(file.Permissions) > 0 {
		isRootOrFileOwner := false
		if u.Username != "root" {
			stats, err := os.Stat(file.Path)
			if err != nil {
				return err
			}
			stat := stats.Sys().(*syscall.Stat_t)
			if u.Uid == strconv.FormatUint(uint64(stat.Uid), 10) && u.Gid == strconv.FormatUint(uint64(stat.Gid), 10) {
				isRootOrFileOwner = true
			}
		} else {
			isRootOrFileOwner = true
		}

		//Fetch permission information
		if isRootOrFileOwner {
			//Make sure agent run as root or file owner
			fileMode, err := strconv.ParseUint(file.Permissions, 8, 32)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("Error parse the file permission %s", file.Permissions))
			}

			err = f.Chmod(fs.FileMode(fileMode))
			if err != nil {
				return err
			}
		} else {
			//Make sure current user is sudoer, and there is "sudo" and "chmod" command in system
			cmd := fmt.Sprintf("sudo chmod %s %s", file.Permissions, file.Path)
			command := exec.Command("/bin/sh", "-c", cmd)
			_, err := command.Output()
			if err != nil {
				return err
			}
		}
	}

	if len(file.Owner) > 0 {

		//Fetch owner information
		if u.Username == "root" {
			//only root can do this
			owner := strings.Split(file.Owner, ":")
			if len(owner) != 2 {
				return errors.Wrap(err, fmt.Sprintf("Invalid owner format '%s'", file.Owner))
			}

			userInfo, err := user.Lookup(owner[0])
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("Error Lookup user %s", owner[0]))
			}

			uid, err := strconv.ParseUint(userInfo.Uid, 10, 32)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("Error convert uid %s", userInfo.Uid))
			}

			gid, err := strconv.ParseUint(userInfo.Gid, 10, 32)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("Error convert gid %s", userInfo.Gid))
			}

			err = f.Chown(int(uid), int(gid))
			if err != nil {
				return err
			}
		} else {
			//Make sure current user is sudoer, and there is "sudo" and "chown" command in system
			cmd := fmt.Sprintf("sudo chown %s %s", file.Owner, file.Path)
			command := exec.Command("/bin/sh", "-c", cmd)
			_, err := command.Output()
			if err != nil {
				return err
			}
		}
	}

	return nil

}
