// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package algo

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
)

const (
	// RockyImgpkgversion defines the imgpkg version that will be installed on host if imgpkg is not already installed
	RockyImgpkgversion = ImgpkgVersion
)

// Rocky8Installer represent the installer implementation for Rock8.* os distribution
type Rocky8Installer struct {
	install   string
	uninstall string
}

// NewRocky8Installer will return new Rocky8Installer instance
func NewRocky8Installer(ctx context.Context, arch, bundleAddrs string) (*Rocky8Installer, error) {
	parseFn := func(script string) (string, error) {
		parser, err := template.New("parser").Parse(script)
		if err != nil {
			return "", fmt.Errorf("unable to parse install script")
		}
		var tpl bytes.Buffer
		if err = parser.Execute(&tpl, map[string]string{
			"BundleAddrs":        bundleAddrs,
			"Arch":               arch,
			"ImgpkgVersion":      RockyImgpkgversion,
			"BundleDownloadPath": "{{.BundleDownloadPath}}",
		}); err != nil {
			return "", fmt.Errorf("unable to apply install parsed template to the data object")
		}
		return tpl.String(), nil
	}

	install, err := parseFn(DoRocky8K8s1_22)
	if err != nil {
		return nil, err
	}
	uninstall, err := parseFn(UndoRocky8K8s1_22)
	if err != nil {
		return nil, err
	}
	return &Rocky8Installer{
		install:   install,
		uninstall: uninstall,
	}, nil
}

// Install will return k8s install script
func (s *Rocky8Installer) Install() string {
	return s.install
}

// Uninstall will return k8s uninstall script
func (s *Rocky8Installer) Uninstall() string {
	return s.uninstall
}

// contains the installation and uninstallation steps for the supported os and k8s
var (
	DoRocky8K8s1_22 = `
set -euox pipefail

BUNDLE_DOWNLOAD_PATH={{.BundleDownloadPath}}
BUNDLE_ADDR={{.BundleAddrs}}
IMGPKG_VERSION={{.ImgpkgVersion}}
ARCH={{.Arch}}
BUNDLE_PATH=$BUNDLE_DOWNLOAD_PATH/$BUNDLE_ADDR


if ! command -v imgpkg >>/dev/null; then
	echo "installing imgpkg"	
	
	if command -v wget >>/dev/null; then
		dl_bin="wget -nv -O-"
	elif command -v curl >>/dev/null; then
		dl_bin="curl -s -L"
	else
		echo "installing curl"
		sudo yum install -y curl
		dl_bin="curl -s -L"
	fi
	
	$dl_bin github.com/vmware-tanzu/carvel-imgpkg/releases/download/$IMGPKG_VERSION/imgpkg-linux-$ARCH > /tmp/imgpkg
	sudo mv /tmp/imgpkg /usr/local/bin/imgpkg
	sudo chmod +x /usr/local/bin/imgpkg
fi

echo "downloading bundle"
mkdir -p $BUNDLE_PATH
/usr/local/bin/imgpkg pull -r -i $BUNDLE_ADDR -o $BUNDLE_PATH


## disable swap
sudo swapoff -a && sudo sed -ri '/\sswap\s/s/^#?/#/' /etc/fstab

## diable selinux
sudo setenforce 0
sudo sed -i 's/^SELINUX=enforcing$/SELINUX=permissive/' /etc/selinux/config

# disable firewall
echo "Stopping and disabling Firewalld."
sudo systemctl stop firewalld || true
sudo systemctl disable firewalld || true

## load kernal modules
sudo modprobe overlay && sudo modprobe br_netfilter

## adding os configuration
tar -C /etc/sysctl.d -xvf "$BUNDLE_PATH/conf.tar" && sudo sysctl --system 

## installing yum-plugin-versionlock
sudo yum install yum-plugin-versionlock -y

## installing rpm packages
sudo yum install "$BUNDLE_PATH/kubernetes-cni.rpm" "$BUNDLE_PATH/kubelet.rpm" -y
sudo yum versionlock kubernetes-cni kubelet
for pkg in cri-tools kubectl kubeadm; do
	sudo yum install "$BUNDLE_PATH/$pkg.rpm" -y
	sudo yum versionlock "$pkg"
done

## intalling containerd
tar -C / -xvf "$BUNDLE_PATH/containerd.tar"

## starting kubelet service
sudo systemctl daemon-reload && systemctl enable kubelet && systemctl start kubelet

## starting containerd service
sudo systemctl daemon-reload && systemctl enable containerd && systemctl start containerd`

	UndoRocky8K8s1_22 = `
set -euox pipefail

BUNDLE_DOWNLOAD_PATH={{.BundleDownloadPath}}
BUNDLE_ADDR={{.BundleAddrs}}
BUNDLE_PATH=$BUNDLE_DOWNLOAD_PATH/$BUNDLE_ADDR

## disabling containerd service
sudo systemctl stop containerd && systemctl disable containerd && systemctl daemon-reload

## removing containerd configurations and cni plugins
sudo rm -rf /opt/cni/ && sudo rm -rf /opt/containerd/ &&  tar tf "$BUNDLE_PATH/containerd.tar" | xargs -n 1 echo '/' | sed 's/ //g'  | grep -e '[^/]$' | xargs rm -f

## removing deb packages
for pkg in kubeadm kubelet kubectl kubernetes-cni cri-tools; do
	sudo yum remove $pkg -y
done

## removing os configuration
tar tf "$BUNDLE_PATH/conf.tar" | xargs -n 1 echo '/etc/sysctl.d' | sed 's/ //g' | grep -e "[^/]$" | xargs rm -f

## remove kernal modules
sudo modprobe -rq overlay && modprobe -r br_netfilter

## enable firewall
echo "Starting and enabling Firewalld."
sudo systemctl start firewalld || true
sudo systemctl enable firewalld || true

## enable selinux
sudo setenforce 1
sudo sed -i 's/^SELINUX=permissive$/SELINUX=enforcing/' /etc/selinux/config

## enable swap
sudo swapon -a && sed -ri '/\sswap\s/s/^#?//' /etc/fstab

rm -rf $BUNDLE_PATH`
)
