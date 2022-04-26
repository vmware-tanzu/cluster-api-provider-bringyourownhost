// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package installer

// contains the installation and uninstallation steps for the supported os and k8s
var (
	DoUbuntu20_4K8s1_22 = `
set -euo pipefail

BUNDLE_PATH=${BUNDLE_PATH:-"/var/lib/byoh/bundles"}

## disable swap
swapoff -a && sed -ri '/\sswap\s/s/^#?/#/' /etc/fstab

## disable firewall
ufw disable

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

BUNDLE_PATH=${BUNDLE_PATH:-"/var/lib/byoh/bundles"}

## enable swap
swapon -a && sed -ri '/\sswap\s/s/^#?//' /etc/fstab

## enable firewall
ufw enable

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
systemctl stop containerd && systemctl disable containerd && systemctl daemon-reload`
)
