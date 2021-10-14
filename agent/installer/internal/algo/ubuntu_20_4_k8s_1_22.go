package algo

import (
	"path/filepath"
)

type Ubuntu_20_4_k8s_1_22 struct {
	BaseK8sInstaller
}

func (u *Ubuntu_20_4_k8s_1_22) swapStep() Step {
	return &ShellStep{
		Desc:    "SWAP",
		DoCmd:   `swapoff -a && sed -ri '/\sswap\s/s/^#?/#/' /etc/fstab`,
		UndoCmd: `swapon -a && sed -ri '/\sswap\s/s/^#?//' /etc/fstab`}
}

func (u *Ubuntu_20_4_k8s_1_22) firewallStep() Step {
	return &ShellStep{
		Desc:    "FIREWALL",
		DoCmd:   "ufw disable",
		UndoCmd: "ufw enable"}
}

func (u *Ubuntu_20_4_k8s_1_22) kernelModsLoadStep() Step {
	return &ShellStep{
		Desc:    "KERNEL MODULES",
		DoCmd:   "modprobe overlay && modprobe br_netfilter",
		UndoCmd: "modprobe -r overlay && modprobe -r br_netfilter"}
}

func (u *Ubuntu_20_4_k8s_1_22) unattendedUpdStep() Step {
	return &ShellStep{
		Desc:    "AUTO OS UPGRADES",
		DoCmd:   "sed -ri 's/1/0/g' /etc/apt/apt.conf.d/20auto-upgrades",
		UndoCmd: "sed -ri 's/0/1/g' /etc/apt/apt.conf.d/20auto-upgrades"}
}

func (u *Ubuntu_20_4_k8s_1_22) osWideCfgUpdateStep(baseInst BaseK8sInstaller) Step {
	confAbsolutePath := filepath.Join(baseInst.BundlePath, "conf.tar")

	return &ShellStep{
		Desc:    "OS CONFIGURATION",
		DoCmd:   "tar -C / -xvf '" + confAbsolutePath + "' && sysctl --system",
		UndoCmd: "tar tf '" + confAbsolutePath + "' | xargs -n 1 echo '/' | sed 's/ //g' | xargs rm -f"}
}

func (u *Ubuntu_20_4_k8s_1_22) criToolsStep(baseInst BaseK8sInstaller) Step {
	return new(AptStep).create(baseInst, "cri-tools.deb")

}

func (u *Ubuntu_20_4_k8s_1_22) criKubernetesStep(baseInst BaseK8sInstaller) Step {
	return new(AptStep).create(baseInst, "kubernetes-cni.deb")
}

func (u *Ubuntu_20_4_k8s_1_22) kubectlStep(baseInst BaseK8sInstaller) Step {
	return new(AptStep).create(baseInst, "kubectl.deb")
}

func (u *Ubuntu_20_4_k8s_1_22) kubeadmStep(baseInst BaseK8sInstaller) Step {
	return new(AptStep).create(baseInst, "kubeadm.deb")
}

func (u *Ubuntu_20_4_k8s_1_22) kubeletStep(baseInst BaseK8sInstaller) Step {
	return new(AptStep).create(baseInst, "kubelet.deb")
}

func (u *Ubuntu_20_4_k8s_1_22) containerdStep(baseInst BaseK8sInstaller) Step {
	containerdAbsPath := filepath.Join(baseInst.BundlePath, "containerd.tar")

	cmdRmDirs := "rm -rf /opt/cni/ && rm -rf /opt/containerd/ && "
	cmdListTar := "tar tf '" + containerdAbsPath + "'"
	cmdConcatPathSlash := " | xargs -n 1 echo '/' | sed 's/ //g'"
	cmdRmFilesOnly := " | xargs rm -f"

	return &ShellStep{
		Desc:    "CONTAINERD",
		DoCmd:   "tar -C / -xvf '" + containerdAbsPath + "'",
		UndoCmd: cmdRmDirs + cmdListTar + cmdConcatPathSlash + cmdRmFilesOnly}
}

func (u *Ubuntu_20_4_k8s_1_22) containerdDaemonStep() Step {
	return &ShellStep{
		Desc:    "CONTAINERD SERVICE",
		DoCmd:   "systemctl daemon-reload && systemctl enable containerd && systemctl start containerd",
		UndoCmd: "systemctl stop containerd && systemctl disable containerd && systemctl daemon-reload"}
}
