package reconciler

import (
	"github.com/go-logr/logr"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/agent/installer"
)

type K8sOptions struct {
	Registry     string
	DownloadPath string
	Logger       logr.Logger
}

func (o *K8sOptions) Install(k8sVersion string) error {
	k8sInstaller, err := installer.New(o.Registry, o.DownloadPath, o.Logger)
	if err != nil {
		return err
	}
	return k8sInstaller.Install(k8sVersion)
}

func (o *K8sOptions) UnInstall(k8sVersion string) error {
	k8sInstaller, err := installer.New(o.Registry, o.DownloadPath, o.Logger)
	if err != nil {
		return err
	}
	return k8sInstaller.Uninstall(k8sVersion)
}
