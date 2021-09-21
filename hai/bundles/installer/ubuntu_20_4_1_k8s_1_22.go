package installer

type Ubuntu_20_4_1_k8s_1_22 struct {
	Ubuntu_20_4_k8s_1_22
}

func (u *Ubuntu_20_4_1_k8s_1_22) kubeadmStep() Step {
	return &ShellStep{
		DoCmd:   "echo Install KubeAdm Ubuntu 20.04.1",
		UndoCmd: "echo Uninstall KubeAdm Ubuntu 20.04.1"}
}

func (u *Ubuntu_20_4_1_k8s_1_22) swapStep() Step {
	return &ShellStep{
		DoCmd:   "echo disabling swap Ubuntu 20.04.1",
		UndoCmd: "echo enabling swap Ubuntu 20.04.1"}
}
