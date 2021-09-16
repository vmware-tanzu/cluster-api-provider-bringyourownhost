package installer

type Ubuntu_20_4_tkg_1_22 struct {
}

func (u *Ubuntu_20_4_tkg_1_22) Install(steps []ShellStep) {
	for _, s := range steps {
		s.Do()
	}
}

func (u *Ubuntu_20_4_tkg_1_22) Uninstall(steps []ShellStep) {
	for _, s := range steps {
		s.Undo()
	}
}

func (u *Ubuntu_20_4_1_tkg_1_22) GetSteps() []ShellStep {
	return []ShellStep{
		u.KubeletStep(),
		u.KubeadmStep(),
		u.ContainerdStep(),
		u.SwapStep(),
		u.FirewallStep()}
}

func (u *Ubuntu_20_4_tkg_1_22) KubeadmStep() ShellStep {
	return ShellStep{
		DoCmd:   "echo Install KubeAdm Ubuntu 20.04",
		UndoCmd: "echo Uninstall KubeAdm Ubuntu 20.04"}
}

func (u *Ubuntu_20_4_tkg_1_22) KubeletStep() ShellStep {
	return ShellStep{
		DoCmd:   "echo Install Kubelet Ubuntu 20.04",
		UndoCmd: "echo Uninstall Kubelet Ubuntu 20.04"}
}

func (u *Ubuntu_20_4_tkg_1_22) ContainerdStep() ShellStep {
	return ShellStep{
		DoCmd:   "echo Install ContainerD Ubuntu 20.04",
		UndoCmd: "echo Uninstall ContainerD Ubuntu 20.04"}
}

func (u *Ubuntu_20_4_tkg_1_22) SwapStep() ShellStep {
	return ShellStep{
		DoCmd:   "echo disabling swap Ubuntu 20.04",
		UndoCmd: "echo enabling swap Ubuntu 20.04"}
}

func (u *Ubuntu_20_4_tkg_1_22) FirewallStep() ShellStep {
	return ShellStep{
		DoCmd:   "echo disabling firewall Ubuntu 20.04",
		UndoCmd: "echo enabling firewall Ubuntu 20.04"}
}
