package hainstaller // TODO change name of package

import (
	"errors"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/cppforlife/go-cli-ui/ui"
	"github.com/k14s/imgpkg/pkg/imgpkg/cmd"
)

// HostaAgentInstaller class
// repoAddr - The address of the repo from which will be downloaded the bundle.
// downloadPath - The location where the bundle will be downloaded.
type HostAgentInstaller struct {
	repoAddr     string
	downloadPath string
}

// Constructor function for the HostAgentInstaller class.
func NewHostAgentInstaller(repoAddr string, downloadPath string) *HostAgentInstaller {
	return &HostAgentInstaller{repoAddr, downloadPath}
}

// Method that checks if a dirrectory exists.
func checkDirExist(dirPath string) bool {
	if fi, err := os.Stat(dirPath); os.IsNotExist(err) || !fi.IsDir() {
		return false
	}
	return true
}
func checkWebAddrReachable(addr string) error {
	resp, err := http.Get(addr)
	if err != nil {
		print(err.Error())
		return err
	} else if int32(resp.StatusCode) != int32(200) {
		return errors.New("Web addres " + addr + " returned response " + resp.Status)
	} else {
		return nil
	}
}

// Method that downloads the bundle from repoAddr to downloadPath.
// This method automatically downloads the given version for the current  linux
// distribution by using helper methods to gather all required  information. If
// the folder where the bundle should be saved does exist the bundle  is  down-
// loaded. Finally the method returns whether the download was successful.
func (hai *HostAgentInstaller) downloadOCIBundle(k8sVer string) error {
	bundleName, err := hai.getBundleName(k8sVer)
	if err != nil {
		return err
	}
	// TODO: Change to real path.
	bundleAddr := hai.repoAddr + "/bundles/" + bundleName

	if !checkDirExist(hai.downloadPath) {
		err := errors.New("Download path does no exist.")
		log.Print(err)
		return err
	}

	err = checkWebAddrReachable(bundleAddr)
	if err != nil {
		return err
	}

	var confUI = ui.NewConfUI(ui.NewNoopLogger())
	defer confUI.Flush()

	imgpkgCmd := cmd.NewDefaultImgpkgCmd(confUI)

	imgpkgCmd.SetArgs([]string{"pull", "--recursive", "-i", bundleAddr + bundleName, "-o", hai.downloadPath})
	err = imgpkgCmd.Execute()

	if err != nil {
		log.Print(err.Error())
		return err
	}
	return nil
}

// Method which installs the downloaded bundle. This is done by executing the
// install.sh shell script of the given version that comes with the bundle.
func (hai *HostAgentInstaller) InstallOCIBundle(k8sVer string, context string) error {
	// TODO: change to real path
	installerPath := hai.downloadPath + "/" + k8sVer + "/installer/install.sh"

	out, err := exec.Command(installerPath, strings.Fields(context)...).Output()

	if err != nil {
		log.Print(err)
		return err
	}

	println(string(out))

	return nil

}

// Method which uninstalls the currently installed bundle. This is done
// by executing the uninstall.sh shell script that comes with the bundle.
func (hai *HostAgentInstaller) Uninstall() error {
	// TODO: change to real path
	uninstallerPath := hai.downloadPath + "/poc-installer/uninstall.sh"

	out, err := exec.Command(uninstallerPath).Output()

	if err != nil {
		log.Print(err)
		return err
	}

	println(string(out))

	return nil

}

// Method which returns the result after executing a command that returns info.
// Exact output format varies between different distributions but the important
// part is the line starting with the string  "Operating system:"  which  shows
// the exact version of the operating  system.  This  information  is  used  to
// identify the correct installer that needs to be used. Also used is the  line
// starting with "Architecture: " to identify whether we need the 32 or 64  bit
// bundle.
//
// Example output for running the command on Ubuntu:
//
//  Static hostname: ubuntu
//        Icon name: computer-vm
//          Chassis: vm
//       Machine ID: 242642b0e734472abaf8c5337e1174c4
//          Boot ID: 181f08d651b76h39be5b138231427c5c
//   Virtualization: vmware
// Operating System: Ubuntu 20.04.3 LTS
//           Kernel: Linux 5.11.0-27-generic
//     Architecture: x86-64
func (hai *HostAgentInstaller) getHostSystemInfo() (string, error) {
	out, err := exec.Command("hostnamectl").Output()

	if err != nil {
		log.Print(err)
		return "", err
	}

	return string(out), nil
}

// Method which takes the information from the getHostSystemInfo function and
// returns only a string containing the exact version of the opretaion system
// running on the host and the required k8s version if they are supported.
//
// Example: Ubuntu_20.04_x64_k8s_1.2.1
func (hai *HostAgentInstaller) getBundleName(k8s string) (string, error) {
	systemInfo, err := hai.getHostSystemInfo()

	if err != nil {
		return "", err
	}

	const strIndicatingOSline string = "Operating System: "

	type Pair struct {
		os  string
		k8s string
	}
	supportedBundles := []Pair{
		{"Ubuntu 20.04", "1.2.1"},
		{"CentOS Linux 7", "1.2.1"}}
	var bundleName string
	for _, p := range supportedBundles {
		if strings.LastIndex(systemInfo, strIndicatingOSline+p.os) != -1 && p.k8s == k8s {
			bundleName = p.os

			if strings.LastIndex(systemInfo, "Architecture: x86-64") != -1 {
				bundleName += "_" + "x64"
			} else {
				bundleName += "_" + "x32"
			}

			bundleName += "_k8s_" + k8s
			break
		}
	}

	if bundleName == "" {
		err := "OS and k8s version not supported."
		log.Print(err)
		return "", errors.New(err)
	}

	bundleName = strings.ReplaceAll(bundleName, " ", "_")

	return bundleName, nil
}
