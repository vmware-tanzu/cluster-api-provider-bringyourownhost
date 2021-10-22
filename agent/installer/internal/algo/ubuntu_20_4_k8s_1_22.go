package algo

import (
	"fmt"
	"path/filepath"
)

type Ubuntu_20_4_k8s_1_22 struct {
	BaseK8sInstaller
	AptStep
}

func (u *Ubuntu_20_4_k8s_1_22) swapStep() Step {
	return &ShellStep{
		BaseK8sInstaller: &u.BaseK8sInstaller,
		Desc:             "SWAP",
		DoCmd:            `swapoff -a && sed -ri '/\sswap\s/s/^#?/#/' /etc/fstab`,
		UndoCmd:          `swapon -a && sed -ri '/\sswap\s/s/^#?//' /etc/fstab`}
}

func (u *Ubuntu_20_4_k8s_1_22) firewallStep() Step {
	return &ShellStep{
		BaseK8sInstaller: &u.BaseK8sInstaller,
		Desc:             "FIREWALL",
		DoCmd:            "ufw disable",
		UndoCmd:          "ufw enable"}
}

func (u *Ubuntu_20_4_k8s_1_22) kernelModsLoadStep() Step {
	return &ShellStep{
		BaseK8sInstaller: &u.BaseK8sInstaller,
		Desc:             "KERNEL MODULES",
		DoCmd:            "modprobe overlay && modprobe br_netfilter",
		UndoCmd:          "modprobe -r overlay && modprobe -r br_netfilter"}
}

func (u *Ubuntu_20_4_k8s_1_22) unattendedUpdStep() Step {
	return &ShellStep{
		BaseK8sInstaller: &u.BaseK8sInstaller,
		Desc:             "AUTO OS UPGRADES",
		DoCmd:            "sed -ri 's/1/0/g' /etc/apt/apt.conf.d/20auto-upgrades",
		UndoCmd:          "sed -ri 's/0/1/g' /etc/apt/apt.conf.d/20auto-upgrades"}
}

func (u *Ubuntu_20_4_k8s_1_22) osWideCfgUpdateStep() Step {
	confAbsolutePath := filepath.Join(u.BundlePath, "conf.tar")

	doCmd := fmt.Sprintf(
		"tar -C / -xvf '%s' && sysctl --system",
		confAbsolutePath)

	undoCmd := fmt.Sprintf(
		"tar tf '%s' | xargs -n 1 echo '/' | sed 's/ //g' | xargs rm -f",
		confAbsolutePath)

	return &ShellStep{
		BaseK8sInstaller: &u.BaseK8sInstaller,
		Desc:             "OS CONFIGURATION",
		DoCmd:            doCmd,
		UndoCmd:          undoCmd}
}

func (u *Ubuntu_20_4_k8s_1_22) criToolsStep() Step {
	return u.NewAptStep(&u.BaseK8sInstaller, "cri-tools.deb")
}

func (u *Ubuntu_20_4_k8s_1_22) criKubernetesStep() Step {
	return u.NewAptStep(&u.BaseK8sInstaller, "kubernetes-cni.deb")
}

func (u *Ubuntu_20_4_k8s_1_22) kubectlStep() Step {
	return u.NewAptStep(&u.BaseK8sInstaller, "kubectl.deb")
}

func (u *Ubuntu_20_4_k8s_1_22) kubeadmStep() Step {
	return u.NewAptStep(&u.BaseK8sInstaller, "kubeadm.deb")
}

func (u *Ubuntu_20_4_k8s_1_22) kubeletStep() Step {
	return u.NewAptStep(&u.BaseK8sInstaller, "kubelet.deb")
}

func (u *Ubuntu_20_4_k8s_1_22) containerdStep() Step {
	containerdAbsPath := filepath.Join(u.BundlePath, "containerd.tar")

	cmdRmDirs := "rm -rf /opt/cni/ && rm -rf /opt/containerd/ && "
	cmdListTar := fmt.Sprintf("tar tf '%s'", containerdAbsPath)
	cmdConcatPathSlash := " | xargs -n 1 echo '/' | sed 's/ //g'"
	cmdRmFilesOnly := " | grep -e '[^/]$' | xargs rm -f"

	doCmd := fmt.Sprintf("tar -C / -xvf '%s'", containerdAbsPath)
	undoCmd := cmdRmDirs + cmdListTar + cmdConcatPathSlash + cmdRmFilesOnly

	return &ShellStep{
		BaseK8sInstaller: &u.BaseK8sInstaller,
		Desc:             "CONTAINERD",
		DoCmd:            doCmd,
		UndoCmd:          undoCmd}
}

func (u *Ubuntu_20_4_k8s_1_22) containerdDaemonStep() Step {
	return &ShellStep{
		BaseK8sInstaller: &u.BaseK8sInstaller,
		Desc:             "CONTAINERD SERVICE",
		DoCmd:            "systemctl daemon-reload && systemctl enable containerd && systemctl start containerd",
		UndoCmd:          "systemctl stop containerd && systemctl disable containerd && systemctl daemon-reload"}
}
