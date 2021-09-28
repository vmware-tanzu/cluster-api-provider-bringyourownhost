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
	systemInfo, err := hai.getHostSystemInfo()
	if err != nil {
		return err
	}
	bundleName, err := hai.getBundleName(systemInfo, k8sVer)
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

// Method which installs the downloaded bundle.
func (hai *HostAgentInstaller) InstallOCIBundle(k8sVer string, context []string) error {
	//placeholder
	/*installer.RunInstaller(
	append([]string{"install", k8sVer}, context...),
	&installer.BaseK8sInstaller{K8sStepProvider: &installer.Ubuntu_20_4_3_k8s_1_22{}})*/
	return nil

}

// Method which uninstalls the currently installed bundle.
func (hai *HostAgentInstaller) UninstallOCIBundle(k8sVer string, context []string) error {
	//placeholder
	/*installer.RunInstaller(
	append([]string{"uninstall", k8sVer}, context...),
	&installer.BaseK8sInstaller{K8sStepProvider: &installer.Ubuntu_20_4_3_k8s_1_22{}})*/
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

// Method which normalizes given os, arch and k8s version to the correct format
func normalizeBundleName(os, arch, k8s string) string {
	bundleName := os
	if arch == "x86-64" {
		bundleName += "_x64"
	} else {
		bundleName += "_x32"
	}
	bundleName += "_k8s_" + k8s

	bundleName = strings.ReplaceAll(bundleName, " ", "_")

	return bundleName
}

// placeholder
type Installer interface {
	install()
	uninstall()
}

func RunInstaller(osArgs []string, i Installer) {
}

type Ubuntu_20_4_3_k8s_1_22 struct {
}

func (u *Ubuntu_20_4_3_k8s_1_22) install() {
}

func (u *Ubuntu_20_4_3_k8s_1_22) uninstall() {
}

type Ubuntu_20_4_k8s_1_22 struct {
}

func (u *Ubuntu_20_4_k8s_1_22) install() {
}

func (u *Ubuntu_20_4_k8s_1_22) uninstall() {
}

// Method which takes the information from the getHostSystemInfo function and
// returns only a string containing the exact version of the opretaion system
// running on the host and the required k8s version if they are supported.
//
// Example: Ubuntu_20.04.3_x64_k8s_1.2.1
func (hai *HostAgentInstaller) getBundleName(systemInfo, k8s string) (string, error) {

	const strIndicatingOSline string = "Operating System: "

	type bundleInfo struct {
		os        string
		arch      string
		k8s       string
		installer Installer
	}
	supportedBundles := []bundleInfo{
		{"Ubuntu 20.04", "x86-64", "1.2.1", &Ubuntu_20_4_k8s_1_22{}},
		{"Ubuntu 20.04.3", "x86-64", "1.2.1", &Ubuntu_20_4_3_k8s_1_22{}},
		{"CentOS Linux 7", "i868", "1.2.1", &Ubuntu_20_4_3_k8s_1_22{}}}

	var bundleName string
	for _, p := range supportedBundles {
		pos := strings.LastIndex(systemInfo, strIndicatingOSline+p.os)
		endPos := pos + len(strIndicatingOSline) + len(p.os)
		if pos != -1 &&
			(systemInfo[endPos] == ' ' || systemInfo[endPos] == '\n') &&
			strings.LastIndex(systemInfo, "Architecture: "+p.arch) != -1 &&
			p.k8s == k8s {
			bundleName = normalizeBundleName(p.os, p.arch, p.k8s)
			break
		}
	}

	if bundleName == "" {
		err := "OS and k8s version not supported."
		log.Print(err)
		return "", errors.New(err)
	}

	return bundleName, nil
}
