// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package algo

// Installer generic installer interface
type Installer interface {
	Install() error
	Uninstall() error
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
5) reload the sysctl with the applied config changes so the changes can take
	effect without restarting
6) disable unattended OS updates
*/

// Step execute/rollback interface
type Step interface {
	do() error
	undo() error
}

// K8sStepProvider steps provider for k8s installer
type K8sStepProvider interface {
	getSteps(*BaseK8sInstaller) []Step

	// os state related steps
	swapStep(*BaseK8sInstaller) Step
	firewallStep(*BaseK8sInstaller) Step
	kernelModsLoadStep(*BaseK8sInstaller) Step

	// packages related steps
	osWideCfgUpdateStep(*BaseK8sInstaller) Step
	criToolsStep(*BaseK8sInstaller) Step
	criKubernetesStep(*BaseK8sInstaller) Step
	containerdStep(*BaseK8sInstaller) Step
	containerdDaemonStep(*BaseK8sInstaller) Step
	kubeadmStep(*BaseK8sInstaller) Step
	kubeletStep(*BaseK8sInstaller) Step
	kubectlStep(*BaseK8sInstaller) Step
}

// BaseK8sInstaller is the default k8s installer implementation
type BaseK8sInstaller struct {
	BundlePath string
	Installer
	K8sStepProvider
	OutputBuilder
}

// Install installation of k8s cluster as per configured steps in the provider
func (b *BaseK8sInstaller) Install() error {
	steps := b.getSteps(b)

	for curStep := 0; curStep < len(steps); curStep++ {
		err := steps[curStep].do()

		if err != nil {
			b.rollback(curStep)
			return err
		}
	}

	return nil
}

// Uninstall uninstallation of k8s cluster as per configured steps in the provider
func (b *BaseK8sInstaller) Uninstall() error {
	lastStepIdx := len(b.getSteps(b)) - 1
	b.rollback(lastStepIdx)

	return nil
}

func (b *BaseK8sInstaller) rollback(currentStep int) {
	steps := b.getSteps(b)

	for ; currentStep >= 0; currentStep-- {
		err := steps[currentStep].undo()

		if err != nil {
			b.OutputBuilder.Err(err.Error())

			/*
				DO NOT break with error (return err) at this point
				this will cause the uninstallation to stop
				and leave leftovers behind
			*/
		}
	}
}

func (b *BaseK8sInstaller) getSteps(bki *BaseK8sInstaller) []Step {
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
		to let kubeadm detect that the default container
		engine is not Docker.
	*/

	var steps = []Step{
		b.swapStep(bki),
		b.firewallStep(bki),
		b.kernelModsLoadStep(bki),
		b.osWideCfgUpdateStep(bki),
		b.criToolsStep(bki),
		b.criKubernetesStep(bki),
		b.containerdStep(bki),
		b.containerdDaemonStep(bki),
		b.kubeletStep(bki),
		b.kubectlStep(bki),
		b.kubeadmStep(bki)}

	return steps
}
