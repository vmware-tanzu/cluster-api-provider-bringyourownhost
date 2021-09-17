package installer

/*
#################################################
# This is a base Kubernetes installer           #
# it should be inherited by step factories      #
# that implement the appropriate method factory #
# overriders.                                   #
#################################################
*/

type Step interface {
	Do()
	Undo()
}

type Installer interface {
	Install()
	Uninstall()
}

type K8sInstaller interface {
	KubeadmStep() Step
	KubeletStep() Step
	ContainerdStep() Step
	SwapStep() Step
	FirewallStep() Step
	GetSteps() []Step
}

type BaseK8sInstaller struct {
	K8sInstaller K8sInstaller
}

func (b *BaseK8sInstaller) Install() {
	for _, s := range b.GetSteps() {
		s.Do()
	}
}

func (b *BaseK8sInstaller) Uninstall() {
	for _, s := range b.GetSteps() {
		s.Undo()
	}
}

func (b *BaseK8sInstaller) GetSteps() []Step {
	var steps = []Step{
		b.K8sInstaller.KubeletStep(),
		b.K8sInstaller.KubeadmStep(),
		b.K8sInstaller.ContainerdStep(),
		b.K8sInstaller.SwapStep(),
		b.K8sInstaller.FirewallStep()}

	return steps
}
