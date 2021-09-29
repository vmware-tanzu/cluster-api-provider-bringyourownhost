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

// Method which downloads and installs the bundle.
func (hai *HostAgentInstaller) Install(k8sVer string, context []string) error {
	reg := newRegistry()
	bd := NewBundleDownloader()
	osd := newOSDetector()

	os, err := osd.detect()
	if err != nil {
		return err
	}

	err = bd.Download(hai.repoAddr, hai.downloadPath, os, k8sVer)
	if err != nil {
		return err
	}

	bundleInstaller, err := reg.getInstaller(os, k8sVer)
	if err != nil {
		return err
	}
	bundleInstaller.install()

	//placeholder
	/*installer.RunInstaller(
	append([]string{"install", k8sVer}, context...),
	&installer.BaseK8sInstaller{K8sStepProvider: bundleInstaller})*/
	return nil

}

// Method which uninstalls the currently installed bundle.
func (hai *HostAgentInstaller) Uninstall(k8sVer string, context []string) error {
	//TODO add check if the given version is installed

	reg := newRegistry()
	osd := newOSDetector()

	os, err := osd.detect()
	if err != nil {
		return err
	}

	bundleInstaller, err := reg.getInstaller(os, k8sVer)
	if err != nil {
		return err
	}

	bundleInstaller.uninstall()
	//placeholder
	/*installer.RunInstaller(
	append([]string{"uninstall", k8sVer}, context...),
	&installer.BaseK8sInstaller{K8sStepProvider: bundleInstaller})*/
	return nil
}

// Constructor function for the HostAgentInstaller class.
func NewHostAgentInstaller(repoAddr string, downloadPath string) *HostAgentInstaller {
	return &HostAgentInstaller{repoAddr, downloadPath}
}

// placeholder/mockup, will be deleted
type Installer interface {
	install()
	uninstall()
}
type Ubuntu_20_4_3_k8s_1_22 struct{}

func (u *Ubuntu_20_4_3_k8s_1_22) install()   {}
func (u *Ubuntu_20_4_3_k8s_1_22) uninstall() {}

// end of mockup

// Struct that contains a map of all supported OS and k8s bundles
type registry struct {
	supportedBundles map[string]map[string]Installer
}

// Constructor for the registry struct
func newRegistry() registry {
	supportedBundles := map[string]map[string]Installer{
		"Ubuntu_20.04.3_x64": {"1.22": &Ubuntu_20_4_3_k8s_1_22{}}}

	return registry{supportedBundles}
}

// Method that checks if the given OS and k8s version are supported and returns
// the installer.
func (r *registry) getInstaller(normalizedOS, normalizedk8s string) (Installer, error) {
	if installer, ok := r.supportedBundles[normalizedOS][normalizedk8s]; ok {
		return installer, nil
	} else {
		err := "Bundle not supported."
		log.Print(err)
		return nil, errors.New(err)
	}
}

// Struct containing all the logic for detecting the OS version.
type oSDetector struct {
}

// Constructor for OSDetector
func newOSDetector() *oSDetector {
	return &oSDetector{}
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
func (osd *oSDetector) getHostSystemInfo() (string, error) {
	out, err := exec.Command("hostnamectl").Output()

	if err != nil {
		log.Print(err)
		return "", err
	}

	return string(out), nil
}

// Method which normalizes given os, arch and k8s version to the correct format.
func (osd *oSDetector) normalizeOsName(os, ver, arch string) string {
	osName := strings.TrimSpace(os) + " " + ver
	if arch == "x86-64" {
		osName += "_x64"
	} else {
		osName += "_x32"
	}

	osName = strings.ReplaceAll(osName, " ", "_")

	return osName
}

// Method that extracts the important information from getHostSystemInfo.
func (osd *oSDetector) filterSystemInfo(systemInfo string) (string, string, string) {
	const strIndicatingOSline string = "Operating System: "
	const strIndicatingArchline string = "Architecture: "
	var os, ver, arch string

	i := strings.LastIndex(systemInfo, strIndicatingOSline) + len(strIndicatingOSline)
	for ; !(systemInfo[i] >= '0' && systemInfo[i] <= '9') && systemInfo[i] != '\n'; i++ {
		os += string(systemInfo[i])
	}
	for ; (systemInfo[i] >= '0' && systemInfo[i] <= '9') || systemInfo[i] == '.'; i++ {
		ver += string(systemInfo[i])
	}
	i = strings.LastIndex(systemInfo, strIndicatingArchline) + len(strIndicatingArchline)
	for ; systemInfo[i] != '\n'; i++ {
		arch += string(systemInfo[i])
	}
	return os, ver, arch
}

//Method that returns the os info in normalized format.
func (osd *oSDetector) detect() (string, error) {
	systemInfo, err := osd.getHostSystemInfo()
	if err != nil {
		return "", err
	}
	os, ver, arch := osd.filterSystemInfo(systemInfo)
	if os == "" || ver == "" || arch == "" {
		err := "Could not detect OS correctly."
		log.Print(err)
		return "", errors.New(err)
	}

	normalizedOS := osd.normalizeOsName(os, ver, arch)
	return normalizedOS, nil
}

// Struct for downloading a bundle
type BundleDownloader struct {
}

// Constructor for BundleDownloader
func NewBundleDownloader() *BundleDownloader {
	return &BundleDownloader{}
}

// Method that checks if a dirrectory exists.
func (bd *BundleDownloader) checkDirExist(dirPath string) bool {
	if fi, err := os.Stat(dirPath); os.IsNotExist(err) || !fi.IsDir() {
		return false
	}
	return true
}

// Method that checks if a web address is reachable.
func (bd *BundleDownloader) checkWebAddrReachable(addr string) error {
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
func (bd *BundleDownloader) Download(repoAddr, downloadPath, osVer, k8sVer string) error {

	// TODO: Change to real path.
	bundleAddr := repoAddr + "/bundles/" + osVer + "/" + k8sVer

	if !bd.checkDirExist(downloadPath) {
		err := errors.New("Download path does no exist.")
		log.Print(err)
		return err
	}

	err := bd.checkWebAddrReachable(bundleAddr)
	if err != nil {
		return err
	}

	var confUI = ui.NewConfUI(ui.NewNoopLogger())
	defer confUI.Flush()

	imgpkgCmd := cmd.NewDefaultImgpkgCmd(confUI)

	imgpkgCmd.SetArgs([]string{"pull", "--recursive", "-i", bundleAddr, "-o", downloadPath})
	err = imgpkgCmd.Execute()

	if err != nil {
		log.Print(err.Error())
		return err
	}
	return nil
}
