// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package algo

import (
	"fmt"
	"path/filepath"
)

// Ubuntu20_4K8s1_22 is the configuration for Ubuntu 20.4.X, K8s 1.22.X extending BaseK8sInstaller
type Ubuntu20_4K8s1_22 struct {
	BaseK8sInstaller
}

func (u *Ubuntu20_4K8s1_22) swapStep(bki *BaseK8sInstaller) Step {
	return &ShellStep{
		BaseK8sInstaller: bki,
		Desc:             "SWAP",
		DoCmd:            `swapoff -a && sed -ri '/\sswap\s/s/^#?/#/' /etc/fstab`,
		UndoCmd:          `swapon -a && sed -ri '/\sswap\s/s/^#?//' /etc/fstab`}
}

func (u *Ubuntu20_4K8s1_22) firewallStep(bki *BaseK8sInstaller) Step {
	return &ShellStep{
		BaseK8sInstaller: bki,
		Desc:             "FIREWALL",
		DoCmd:            "ufw disable",
		UndoCmd:          "ufw enable"}
}

func (u *Ubuntu20_4K8s1_22) kernelModsLoadStep(bki *BaseK8sInstaller) Step {
	return &ShellStep{
		BaseK8sInstaller: bki,
		Desc:             "KERNEL MODULES",
		DoCmd:            "modprobe overlay && modprobe br_netfilter",
		UndoCmd:          "modprobe -r overlay && modprobe -r br_netfilter"}
}

func (u *Ubuntu20_4K8s1_22) osWideCfgUpdateStep(bki *BaseK8sInstaller) Step {
	confAbsolutePath := filepath.Join(bki.BundlePath, "conf.tar")

	doCmd := fmt.Sprintf(
		"tar -C / -xvf '%s' && sysctl --system",
		confAbsolutePath)

	undoCmd := fmt.Sprintf(
		"tar tf '%s' | xargs -n 1 echo '/' | sed 's/ //g' | xargs rm -f",
		confAbsolutePath)

	return &ShellStep{
		BaseK8sInstaller: bki,
		Desc:             "OS CONFIGURATION",
		DoCmd:            doCmd,
		UndoCmd:          undoCmd}
}

func (u *Ubuntu20_4K8s1_22) criToolsStep(bki *BaseK8sInstaller) Step {
	// Not available upstream
	return NewAptStepOptional(bki, "cri-tools.deb")
}

func (u *Ubuntu20_4K8s1_22) criKubernetesStep(bki *BaseK8sInstaller) Step {
	// Not available upstream
	return NewAptStepOptional(bki, "kubernetes-cni.deb")
}

func (u *Ubuntu20_4K8s1_22) kubectlStep(bki *BaseK8sInstaller) Step {
	return NewAptStep(bki, "kubectl.deb")
}

func (u *Ubuntu20_4K8s1_22) kubeadmStep(bki *BaseK8sInstaller) Step {
	return NewAptStep(bki, "kubeadm.deb")
}

func (u *Ubuntu20_4K8s1_22) kubeletStep(bki *BaseK8sInstaller) Step {
	return NewAptStep(bki, "kubelet.deb")
}

func (u *Ubuntu20_4K8s1_22) containerdStep(bki *BaseK8sInstaller) Step {
	containerdAbsPath := filepath.Join(bki.BundlePath, "containerd.tar")

	cmdRmDirs := "rm -rf /opt/cni/ && rm -rf /opt/containerd/ && "
	cmdListTar := fmt.Sprintf("tar tf '%s'", containerdAbsPath)
	cmdConcatPathSlash := " | xargs -n 1 echo '/' | sed 's/ //g'"
	cmdRmFilesOnly := " | grep -e '[^/]$' | xargs rm -f"

	doCmd := fmt.Sprintf("tar -C / -xvf '%s'", containerdAbsPath)
	undoCmd := cmdRmDirs + cmdListTar + cmdConcatPathSlash + cmdRmFilesOnly

	return &ShellStep{
		BaseK8sInstaller: bki,
		Desc:             "CONTAINERD",
		DoCmd:            doCmd,
		UndoCmd:          undoCmd}
}

func (u *Ubuntu20_4K8s1_22) containerdDaemonStep(bki *BaseK8sInstaller) Step {
	return &ShellStep{
		BaseK8sInstaller: bki,
		Desc:             "CONTAINERD SERVICE",
		DoCmd:            "systemctl daemon-reload && systemctl enable containerd && systemctl start containerd",
		UndoCmd:          "systemctl stop containerd && systemctl disable containerd && systemctl daemon-reload"}
}
