package installer

import (
	"github.com/go-logr/logr"
)

type Error string
func (e Error) Error() string { return string(e) }

const (
	ErrDetectOs		= Error("Error detecting OS")
	ErrOsK8sNotSupported	= Error("No k8s support for OS")
	ErrBundleDownload       = Error("Error downloading bundle")
	ErrBundleExtract        = Error("Error extracting bundle")
	ErrBundleInstall        = Error("Error installing bundle")
	ErrBundleUninstall      = Error("Error uninstalling bundle")
)

type installer struct {
	bundleRepo string
	downloadPath string
	logger logr.Logger
	algoRegistry registry
	osDetector
	bundleDownloader
}

func getAlgoRegistry(downloadPath string) registry {
	bd := bundleDownloader{}
	reg := NewRegistry()

	var os, k8s string
        os, k8s = "ubuntu", "1_22"; reg.Add(os, k8s, bd.getK8sDirPath(downloadPath, k8s))
	// To add support for new os or k8s add here

	return reg
}

func New(bundleRepo, downloadPath string, logger logr.Logger) (*installer, error) {
	reg := getAlgoRegistry(downloadPath)

	osDetector := osDetector{}
	if os, err := osDetector.Detect(); err != nil {
		return nil, ErrDetectOs
	} else {
		if len(reg.ListK8s(os)) == 0 {
                        return nil, ErrOsK8sNotSupported
                }
	}

	return &installer{bundleRepo : bundleRepo,
	                  downloadPath : downloadPath,
	                  logger : logger,
			  algoRegistry : reg,
			  osDetector : osDetector}, nil
}

func (i *installer) Install(k8sVer string) error {
	_, err := i.getAlgoInstallerWithBundle(k8sVer)
	if err != nil {
		return err
	}
	//err = algoInst.Install
	if err != nil {
                return ErrBundleInstall
        }

	return err
}

func (i *installer) Uninstall(k8sVer string) error {
	_, err := i.getAlgoInstallerWithBundle(k8sVer)
        if err != nil {
                return err
        }
        //err = algoInst.Uninstall
	if err != nil {
		return ErrBundleUninstall
	}

        return nil
}

func (i *installer) getAlgoInstallerWithBundle(k8sVer string) (osk8sInstaller, error) {
	os,_ := i.osDetector.Detect()
        algoInst := i.algoRegistry.GetInstaller(os, k8sVer)
        if algoInst != nil {
                return nil, ErrOsK8sNotSupported
        }

        bdErr := i.bundleDownloader.Download(i.bundleRepo, i.downloadPath, os, k8sVer)
        if bdErr != nil {
                return nil, ErrBundleDownload
        }

	return algoInst, nil
}

func ListSupportedOS() []string {
	reg := getAlgoRegistry("")
	return reg.ListOS()
}

func ListSupportedK8s(os string) ([]string, error) {
	reg := getAlgoRegistry("")
	return reg.ListK8s(os), nil
	// TODO error?
}

// PreviewChanges describes the changes to install and uninstall K8s on OS without actually applying them.
// It returns the install and uninstall changes
func PreviewChanges(os, k8sVer string) (install, uninstall string, err error) {
	return "","", nil
}
