package installer

type Ubuntu_20_4_1_tkg_1_22 struct {
	Ubuntu_20_4_tkg_1_22
}

func (u *Ubuntu_20_4_1_tkg_1_22) KubeadmStep() ShellStep {
	return ShellStep{
		DoCmd:   "echo Install KubeAdm Ubuntu 20.04.1",
		UndoCmd: "echo Uninstall KubeAdm Ubuntu 20.04.1"}
}

func (u *Ubuntu_20_4_1_tkg_1_22) SwapStep() ShellStep {
	return ShellStep{
		DoCmd:   "echo disabling swap Ubuntu 20.04.1",
		UndoCmd: "echo enabling swap Ubuntu 20.04.1"}
}
