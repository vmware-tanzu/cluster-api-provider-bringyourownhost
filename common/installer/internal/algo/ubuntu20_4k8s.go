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
	// ImgpkgVersion defines the imgpkg version that will be installed on host if imgpkg is not already installed
	ImgpkgVersion = "v0.27.0"
)

// Ubuntu20_04Installer represent the installer implementation for ubunto20.04.* os distribution
type Ubuntu20_04Installer struct {
	install   string
	uninstall string
}

// NewUbuntu20_04Installer will return new Ubuntu20_04Installer instance
func NewUbuntu20_04Installer(ctx context.Context, arch, bundleAddrs string) (*Ubuntu20_04Installer, error) {
	parseFn := func(script string) (string, error) {
		parser, err := template.New("parser").Parse(script)
		if err != nil {
			return "", fmt.Errorf("unable to parse install script")
		}
		var tpl bytes.Buffer
		if err = parser.Execute(&tpl, map[string]string{
			"BundleAddrs":        bundleAddrs,
			"Arch":               arch,
			"ImgpkgVersion":      ImgpkgVersion,
			"BundleDownloadPath": "{{.BundleDownloadPath}}",
		}); err != nil {
			return "", fmt.Errorf("unable to apply install parsed template to the data object")
		}
		return tpl.String(), nil
	}

	install, err := parseFn(DoUbuntu20_4K8s1_22)
	if err != nil {
		return nil, err
	}
	uninstall, err := parseFn(UndoUbuntu20_4K8s1_22)
	if err != nil {
		return nil, err
	}
	return &Ubuntu20_04Installer{
		install:   install,
		uninstall: uninstall,
	}, nil
}

// Install will return k8s install script
func (s *Ubuntu20_04Installer) Install() string {
	return s.install
}

// Uninstall will return k8s uninstall script
func (s *Ubuntu20_04Installer) Uninstall() string {
	return s.uninstall
}

// contains the installation and uninstallation steps for the supported os and k8s
var (
	DoUbuntu20_4K8s1_22 = `
set -euo pipefail

BUNDLE_DOWNLOAD_PATH={{.BundleDownloadPath}}
BUNDLE_ADDR={{.BundleAddrs}}
IMGPKG_VERSION={{.ImgpkgVersion}}
ARCH={{.Arch}}
BUNDLE_PATH=$BUNDLE_DOWNLOAD_PATH/$BUNDLE_ADDR


if ! command -v imgpkg >>/dev/null; then
	echo "installing imgpkg"
	wget -nv -O- github.com/vmware-tanzu/carvel-imgpkg/releases/download/$IMGPKG_VERSION/imgpkg-linux-$ARCH > /tmp/imgpkg
	mv /tmp/imgpkg /usr/local/bin/imgpkg
	chmod +x /usr/local/bin/imgpkg
fi

<<<<<<< HEAD
echo "downloading bundle"
mkdir -p $BUNDLE_PATH
imgpkg pull -r -i $BUNDLE_ADDR -o $BUNDLE_PATH
=======
if [ ! -d $BUNDLE_PATH ]; then
	echo "downloading bundle"
	mkdir -p $BUNDLE_PATH
	imgpkg pull -r -i $BUNDLE_ADDR -o $BUNDLE_PATH
fi
>>>>>>> host agent changes added


## disable swap
swapoff -a && sed -ri '/\sswap\s/s/^#?/#/' /etc/fstab

## disable firewall
if command -v ufw >>/dev/null; then
	ufw disable
fi

## load kernal modules
modprobe overlay && modprobe br_netfilter

## adding os configuration
tar -C / -xvf "$BUNDLE_PATH/conf.tar" && sysctl --system 

## installing deb packages
for pkg in cri-tools kubernetes-cni kubectl kubeadm kubelet; do
	dpkg --install "$BUNDLE_PATH/$pkg.deb" && apt-mark hold $pkg
done

## intalling containerd
tar -C / -xvf "$BUNDLE_PATH/containerd.tar"

## starting containerd service
systemctl daemon-reload && systemctl enable containerd && systemctl start containerd`

	UndoUbuntu20_4K8s1_22 = `
set -euo pipefail

BUNDLE_DOWNLOAD_PATH={{.BundleDownloadPath}}
BUNDLE_ADDR={{.BundleAddrs}}
BUNDLE_PATH=$BUNDLE_DOWNLOAD_PATH/$BUNDLE_ADDR

## enable swap
swapon -a && sed -ri '/\sswap\s/s/^#?//' /etc/fstab

## enable firewall
if command -v ufw >>/dev/null; then
	ufw enable
fi

## remove kernal modules
modprobe -r overlay && modprobe -r br_netfilter

## removing os configuration
tar tf "$BUNDLE_PATH/conf.tar" | xargs -n 1 echo '/' | sed 's/ //g' | xargs rm -f

## removing deb packages
for pkg in cri-tools kubernetes-cni kubectl kubeadm kubelet; do
	dpkg --purge $pkg
done

## removing containerd configurations and cni plugins
rm -rf /opt/cni/ && rm -rf /opt/containerd/ &&  tar tf "$BUNDLE_PATH/containerd.tar" | xargs -n 1 echo '/' | sed 's/ //g'  | grep -e '[^/]$' | xargs rm -f

## disabling containerd service
systemctl stop containerd && systemctl disable containerd && systemctl daemon-reload

rm -rf $BUNDLE_PATH`
)
