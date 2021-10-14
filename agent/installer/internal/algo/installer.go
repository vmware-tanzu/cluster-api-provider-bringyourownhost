package algo

type OutputBuilder interface {
	Description(string)
	Command(string)
	CommandOutput(string)
}

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
*/

type Step interface {
	do(k *BaseK8sInstaller) error
	undo(k *BaseK8sInstaller) error
}

/*
##################
# IMPORTANT NOTE #
##################

Please note that the following steps:

	kernelModsLoadStep() Step
	kernelModsCfgStep() Step
	netForwardingStep() Step
	sysctlReloadStep() Step

are required in order to:
1) enable the kernel modules: overlay & bridge network filter
2) create a systemwide config that will enable those modules on boot time
3) enable ipv4 & ipv6 forwarding
4) create a systemwide config that will enable the forwarding on boot time
5) realod the sysctl with the applied config changes so the changes can take
   effect without restarting
6) disable unattended OS updates
*/

type K8sStepProvider interface {
	getSteps() []Step

	swapStep() Step
	firewallStep() Step
	unattendedUpdStep() Step
	kernelModsLoadStep() Step

	osWideCfgUpdateStep(baseInst BaseK8sInstaller) Step
	criToolsStep(baseInst BaseK8sInstaller) Step
	criKubernetesStep(baseInst BaseK8sInstaller) Step
	containerdStep(baseInst BaseK8sInstaller) Step
	containerdDaemonStep() Step
	kubeadmStep(baseInst BaseK8sInstaller) Step
	kubeletStep(baseInst BaseK8sInstaller) Step
	kubectlStep(baseInst BaseK8sInstaller) Step
}

//This is the default k8s installer implementation
type BaseK8sInstaller struct {
	BundlePath string
	Installer
	K8sStepProvider
	LogBuilder
}

func (b *BaseK8sInstaller) install() error {
	steps := b.getSteps()

	for curStep := 0; curStep < len(steps); curStep++ {
		err := steps[curStep].do(b)

		if err != nil {
			b.LogBuilder.AddTimestamp(&b.LogBuilder.stdErr).
				AddStdErr("*** CRITICAL ERROR ***").
				AddStdErr(err.Error())

			b.rollBackInstallation(curStep)
			return err
		}
	}

	return nil
}

func (b *BaseK8sInstaller) rollBackInstallation(curStep int) {
	b.LogBuilder.
		AddTimestamp(&b.LogBuilder.stdOut).
		AddStdOut("*** UNDOING PARTIAL INSTALLATION...")

	steps := b.getSteps()

	for ; curStep >= 0; curStep-- {
		steps[curStep].undo(b)
	}
}

func (b *BaseK8sInstaller) uninstall() error {
	steps := b.getSteps()
	stepsCnt := len(steps)

	for curStep := stepsCnt - 1; curStep >= 0; curStep-- {
		err := steps[curStep].undo(b)

		if err != nil {
			b.LogBuilder.
				AddTimestamp(&b.LogBuilder.stdErr).
				AddStdErr(err.Error())

			return err
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
		b.K8sStepProvider.swapStep(),
		b.K8sStepProvider.firewallStep(),
		b.K8sStepProvider.unattendedUpdStep(),
		b.K8sStepProvider.kernelModsLoadStep(),
		b.K8sStepProvider.osWideCfgUpdateStep(*b),
		b.K8sStepProvider.criToolsStep(*b),
		b.K8sStepProvider.criKubernetesStep(*b),
		b.K8sStepProvider.containerdStep(*b),
		b.K8sStepProvider.containerdDaemonStep(),
		b.K8sStepProvider.kubeletStep(*b),
		b.K8sStepProvider.kubectlStep(*b),
		b.K8sStepProvider.kubeadmStep(*b)}

	return steps
}
