package algo

// This is a generic installer interface
type Installer interface {
	install() error
	uninstall() error
}

/*
#################################################
# This is a Kubernetes installer step provider  #
# it should be inherited by step factories      #
# that implement the appropriate method factory #
# overriders.                                   #
#################################################

** IMPORTANT NOTE **

Please note that the following steps:

	kernelModsLoadStep() Step
	kernelModsCfgStep() Step
	netForwardingStep() Step
	sysctlReloadStep() Step

are required in order to:
1) enable the kernel modules: overlay & bridge network filter
2) create a systemwide config that will enable those modules at boot time
3) enable ipv4 & ipv6 forwarding
4) create a systemwide config that will enable the forwarding at boot time
5) realod the sysctl with the applied config changes so the changes can take
   effect without restarting
6) disable unattended OS updates
*/

type Step interface {
	do() error
	undo() error
}

type K8sStepProvider interface {
	getSteps() []Step

	//os state related steps
	swapStep() Step
	firewallStep() Step
	unattendedUpdStep() Step
	kernelModsLoadStep() Step

	//packages related steps
	osWideCfgUpdateStep() Step
	criToolsStep() Step
	criKubernetesStep() Step
	containerdStep() Step
	containerdDaemonStep() Step
	kubeadmStep() Step
	kubeletStep() Step
	kubectlStep() Step
}

//This is the default k8s installer implementation
type BaseK8sInstaller struct {
	BundlePath string
	Installer
	K8sStepProvider
	OutputBuilder
}

func (b *BaseK8sInstaller) install() error {
	steps := b.getSteps()

	for curStep := 0; curStep < len(steps); curStep++ {
		err := steps[curStep].do()

		if err != nil {
			b.OutputBuilder.Msg("Critical error.")
			b.OutputBuilder.StdErr(err.Error())
			b.rollBackInstallation(curStep)
			return err
		}
	}

	return nil
}

func (b *BaseK8sInstaller) rollBackInstallation(curStep int) {
	steps := b.getSteps()

	for ; curStep >= 0; curStep-- {
		steps[curStep].undo()
	}
}

func (b *BaseK8sInstaller) uninstall() error {
	steps := b.getSteps()
	stepsCnt := len(steps)

	for curStep := stepsCnt - 1; curStep >= 0; curStep-- {
		err := steps[curStep].undo()

		if err != nil {
			b.OutputBuilder.StdErr(err.Error())

			//DO NOT break with error (return err) at this point
			//this will cause the uninstallation to stop
			//and leave leftovers behind
		}
	}

	return nil
}

func (b *BaseK8sInstaller) getSteps() []Step {
	/*
		##################
		# IMPORTANT NOTE #
		##################
		Order of execution matters!

		For instance some packages are dependent on the
		CRI-Tools & CRI-Kubernetes-CNI
		Others have to be installed after kubectl.

		Kernel modules have to be loaded and configured
		and ip forwarding has to be enabled
		prior to start working with kubeadm.

		ContainerD has to be loaded as a daemon first, in order
		to let kubeadm to detect that the default container
		engine is not Docker.
	*/

	var steps = []Step{
		b.swapStep(),
		b.firewallStep(),
		b.unattendedUpdStep(),
		b.kernelModsLoadStep(),
		b.osWideCfgUpdateStep(),
		b.criToolsStep(),
		b.criKubernetesStep(),
		b.containerdStep(),
		b.containerdDaemonStep(),
		b.kubeletStep(),
		b.kubectlStep(),
		b.kubeadmStep()}

	return steps
}
