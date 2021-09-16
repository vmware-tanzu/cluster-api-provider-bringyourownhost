package installer

/*
#################################################
# This is a base Kubernetes installer           #
# it should be inherited by step factories      #
# that implement the appropriate method factory #
# overriders.                                   #
#################################################
*/
type BaseK8sInstaller interface {
	Install()
	Uninstall()
	KubeadmStep() Step
	KubeletStep() Step
	ContainerdStep() Step
	SwapStep() Step
	FirewallStep() Step
	GetSteps(steps []Step) []Step
}

type Step interface {
	Do()
	Undo()
}
