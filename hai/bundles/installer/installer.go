package installer

// This is a generic installer interface
type Installer interface {
	install()
	uninstall()
}

/*
#################################################
# This is a Kubernetes installer step provider  #
# it should be inherited by step factories      #
# that implement the appropriate method factory #
# overriders.                                   #
#################################################
*/

type Step interface {
	do()
	undo()
}

type K8sStepProvider interface {
	kubeadmStep() Step
	kubeletStep() Step
	containerdStep() Step
	swapStep() Step
	firewallStep() Step
	getSteps() []Step
}

//This is the default k8s installer implementation
type BaseK8sInstaller struct {
	Installer
	K8sStepProvider
}

func (b *BaseK8sInstaller) install() {
	for _, s := range b.getSteps() {
		s.do()
	}
}

func (b *BaseK8sInstaller) uninstall() {
	for _, s := range b.getSteps() {
		s.undo()
	}
}

func (b *BaseK8sInstaller) getSteps() []Step {
	var steps = []Step{
		b.K8sStepProvider.kubeletStep(),
		b.K8sStepProvider.kubeadmStep(),
		b.K8sStepProvider.containerdStep(),
		b.K8sStepProvider.swapStep(),
		b.K8sStepProvider.firewallStep()}

	return steps
}
