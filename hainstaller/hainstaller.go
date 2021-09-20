package main // TODO change name of package

import (
	"log"
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

// Constructor function for the HostAgentInstaller class
//
// @param repoAddr - string
// @param downloadPath - string
//
// @return *HostAgentInstaller
func NewHostAgentInstaller(repoAddr string, downloadPath string) *HostAgentInstaller {
	return &HostAgentInstaller{repoAddr, downloadPath}
}

// Method that downloads the bundle from repoAddr to downloadPath.
// This method automatically downloads the given version for the current  linux
// distribution by using helper methods to gather all required information. The
// folder where the bundle should be saved is created recursively  if  it  does
// not exist and then the download is being performed using the carvel  imgpkg.
// Finally the method returns the output of the executed command.
//
// @param k8sVer - string
//
// @return []byte
func (hai *HostAgentInstaller) Download(k8sVer string) []byte {
	osInfo := hai.GetHostOS()

	// TODO: Change to real path.
	bundleName := osInfo + "_TKG_" + k8sVer
	bundleAddr := hai.repoAddr + "/bundles/" + bundleName

	// Check if the folder downloadPath exists.
	// If it does not, it is being created recursively.
	if fi, err := os.Stat(hai.downloadPath); os.IsNotExist(err) || !fi.IsDir() {

		_, err := exec.Command("mkdir", "-p", hai.downloadPath).Output()

		if err != nil {
			log.Fatal(err)
		}
	}
	var confUI = ui.NewConfUI(ui.NewNoopLogger())
	defer confUI.Flush()

	imgpkgCmd := cmd.NewDefaultImgpkgCmd(confUI)

	//used for debugging
	//bundleAddr = "projects.registry.vmware.com"
	//bundleName = "/cluster_api_provider_byoh/hello-world:latest"

	imgpkgCmd.SetArgs([]string{"pull", "--recursive", "-i", bundleAddr + bundleName, "-o", hai.downloadPath})
	err := imgpkgCmd.Execute()

	if err != nil {
		log.Fatal(err.Error())
	}
	return ([]byte("Done"))
}

// Method which installs the downloaded bundle. This is done by executing the
// install.sh shell script of the given version that comes with the bundle.
//
// @param k8sVer - String indicating the k8s version that needs to be installed
// @param context - additional arguments passed to the installer
//
// return []byte - the output of the executed command
func (hai *HostAgentInstaller) Install(k8sVer string, context string) []byte {
	// TODO: change to real path
	installerPath := hai.downloadPath + "/" + k8sVer + "/installer/install.sh"

	out, err := exec.Command(installerPath, strings.Fields(context)...).Output()

	if err != nil {
		log.Fatal(err)
	}

	println(string(out))

	return out

}

// Method which uninstalls the currently installed bundle. This is done
// by executing the uninstall.sh shell script that comes with the bundle.
//
// return []byte - the output of the executed command
func (hai *HostAgentInstaller) Uninstall() []byte {
	// TODO: change to real path
	uninstallerPath := hai.downloadPath + "/poc-installer/uninstall.sh"

	out, err := exec.Command(uninstallerPath).Output()

	if err != nil {
		log.Fatal(err)
	}

	println(string(out))

	return out

}

// Method which returns the result after executing  the  command  "hostnamectl"
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
//
//
// @return string
func (hai *HostAgentInstaller) getHostSystemInfo() string {
	out, err := exec.Command("hostnamectl").Output()

	if err != nil {
		log.Fatal(err)
	}

	return string(out)
}

// Method which takes the information from the getHostSystemInfo function and
// returns only a string containing the exact version of the opretaion system
// running on the host.
//
// Example: Ubuntu_x64_20
//			Red_Hat_Enterprise_Linux_x64_20
//
// @retrun string
func (hai *HostAgentInstaller) GetHostOS() string {
	systemInfo := hai.getHostSystemInfo()

	// The string which is indicating the OS and its version
	const strIndicatingOSline string = "Operating System: "

	var osInfo string
	var pos int
	for pos = strings.LastIndex(systemInfo, strIndicatingOSline) +
		len(strIndicatingOSline); pos < len(systemInfo); pos++ {

		char := systemInfo[pos]
		if !((char >= 'A' && char <= 'Z') ||
			(char >= 'a' && char <= 'z') ||
			char == ' ') {
			break
		}
		if char == ' ' {
			osInfo += "_"
		} else {
			osInfo += string(char)
		}
	}
	osInfo = osInfo[:len(osInfo)-1]

	if osInfo == "" {
		log.Fatal("OS not supported")
	}

	if strings.LastIndex(systemInfo, "Architecture: x86-64") != -1 {
		osInfo += "_x64"
	} else {
		osInfo += "_x32"
	}

	var version string

	for ; pos != '\n' && systemInfo[pos] >= '0' && systemInfo[pos] <= '9'; pos++ {
		version += (string)(systemInfo[pos])
	}

	osInfo += "_" + version

	return osInfo
}
