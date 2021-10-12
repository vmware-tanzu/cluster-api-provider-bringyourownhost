package installer

import (
	"log"
)

type Error string
func (e Error) Error() string { return string(e) }

const (
	ErrorDetectOs		= Error("Error detecting OS")
	ErrorOsK8sNotSupported	= Error("No k8s support for this OS")
	ErrorBundleDownload     = Error("Error downloading bundle")
	ErrorBundleExtract      = Error("Error extracting bundle")
	ErrorBundleInstall      = Error("Error installing bundle")
	ErrorBundleUninstall    = Error("Error uninstalling bundle")
)

type installer struct {
	bundleRepo string
	downloadPath string
	logger *log.Logger
}

func New(bundleRepo, downloadPath string, logger *log.Logger) (*installer, error) {
	return &installer{bundleRepo : bundleRepo,
	                  downloadPath : downloadPath,
	                  logger :logger  }, ErrorOsK8sNotSupported
}

func (i *installer) Install(k8sVer string) error {
	return ErrorOsK8sNotSupported
}

func (i *installer) Uninstall(k8sVer string) error {
	return ErrorOsK8sNotSupported
}

func ListSupportedOS() []string {
	return nil
}

func ListSupportedK8s(os string) ([]string, error) {
	return nil, ErrorOsK8sNotSupported
}

func PreviewChanges(os, k8sVer string) (install, uninstall string, err error) {
	return "","", ErrorOsK8sNotSupported
}
