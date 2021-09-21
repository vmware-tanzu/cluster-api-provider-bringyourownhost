package installer

type Ubuntu_20_4_k8s_1_22 struct {
	BaseK8sInstaller
}

func (u *Ubuntu_20_4_k8s_1_22) kubeadmStep() Step {
	return &ShellStep{
		DoCmd:   "echo Install KubeAdm Ubuntu 20.04",
		UndoCmd: "echo Uninstall KubeAdm Ubuntu 20.04"}
}

func (u *Ubuntu_20_4_k8s_1_22) kubeletStep() Step {
	return &ShellStep{
		DoCmd:   "echo Install Kubelet Ubuntu 20.04",
		UndoCmd: "echo Uninstall Kubelet Ubuntu 20.04"}
}

func (u *Ubuntu_20_4_k8s_1_22) containerdStep() Step {
	return &ShellStep{
		DoCmd:   "echo Install ContainerD Ubuntu 20.04",
		UndoCmd: "echo Uninstall ContainerD Ubuntu 20.04"}
}

func (u *Ubuntu_20_4_k8s_1_22) swapStep() Step {
	return &ShellStep{
		DoCmd:   "echo disabling swap Ubuntu 20.04",
		UndoCmd: "echo enabling swap Ubuntu 20.04"}
}

func (u *Ubuntu_20_4_k8s_1_22) firewallStep() Step {
	return &ShellStep{
		DoCmd:   "echo disabling firewall Ubuntu 20.04",
		UndoCmd: "echo enabling firewall Ubuntu 20.04"}
}
