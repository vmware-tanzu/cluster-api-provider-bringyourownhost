// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package algo

type MockUbuntuWithError struct {
	BaseK8sInstaller
}

func getEmptyStep(desc string, bki *BaseK8sInstaller) Step {
	return &ShellStep{
		BaseK8sInstaller: bki,
		Desc:             desc,
		DoCmd:            `:`,
		UndoCmd:          `:`}
}

func (u *MockUbuntuWithError) swapStep(bki *BaseK8sInstaller) Step {
	return getEmptyStep("SWAP", bki)
}

func (u *MockUbuntuWithError) firewallStep(bki *BaseK8sInstaller) Step {
	return getEmptyStep("FIREWALL", bki)
}

func (u *MockUbuntuWithError) kernelModsLoadStep(bki *BaseK8sInstaller) Step {
	return getEmptyStep("KERNEL MODULES", bki)
}

func (u *MockUbuntuWithError) unattendedUpdStep(bki *BaseK8sInstaller) Step {
	return getEmptyStep("AUTO OS UPGRADES", bki)
}

func (u *MockUbuntuWithError) osWideCfgUpdateStep(bki *BaseK8sInstaller) Step {
	// Here DoCmd should return error when executed
	return &ShellStep{
		BaseK8sInstaller: bki,
		Desc:             "OS CONFIGURATION",
		DoCmd:            `a`,
		UndoCmd:          `:`}
}

func (u *MockUbuntuWithError) criToolsStep(bki *BaseK8sInstaller) Step {
	return getEmptyStep("cri-tools.deb", bki)
}

func (u *MockUbuntuWithError) criKubernetesStep(bki *BaseK8sInstaller) Step {
	return getEmptyStep("kubernetes-cni.deb", bki)
}

func (u *MockUbuntuWithError) kubectlStep(bki *BaseK8sInstaller) Step {
	return getEmptyStep("kubectl.deb", bki)
}

func (u *MockUbuntuWithError) kubeadmStep(bki *BaseK8sInstaller) Step {
	return getEmptyStep("kubeadm.deb", bki)
}

func (u *MockUbuntuWithError) kubeletStep(bki *BaseK8sInstaller) Step {
	return getEmptyStep("kubelet.deb", bki)
}

func (u *MockUbuntuWithError) containerdStep(bki *BaseK8sInstaller) Step {
	return getEmptyStep("CONTAINERD", bki)
}

func (u *MockUbuntuWithError) containerdDaemonStep(bki *BaseK8sInstaller) Step {
	return getEmptyStep("CONTAINERD SERVICE", bki)
}
