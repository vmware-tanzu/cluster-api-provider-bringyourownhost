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
}

func New(bundleRepo, downloadPath string, logger logr.Logger) (*installer, error) {
	return &installer{bundleRepo : bundleRepo,
	                  downloadPath : downloadPath,
	                  logger : logger}, nil
}

func (i *installer) Install(k8sVer string) error {
	return nil
}

func (i *installer) Uninstall(k8sVer string) error {
	return nil
}

func ListSupportedOS() []string {
	return nil
}

func ListSupportedK8s(os string) ([]string, error) {
	return nil, nil
}

// PreviewChanges describes the changes to install and uninstall K8s on OS without actually applying them.
// It returns the install and uninstall changes
func PreviewChanges(os, k8sVer string) (install, uninstall string, err error) {
	return "","", nil
}
