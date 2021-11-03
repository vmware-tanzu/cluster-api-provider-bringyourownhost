// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package algo

import "errors"

type mockStep struct {
	Step
	Desc    string
	Err     error
	doCnt   *int
	undoCnt *int
	*BaseK8sInstaller
}

func (s *mockStep) do() error {
	s.OutputBuilder.Msg("Installing: " + s.Desc)
	*s.doCnt++
	return s.Err
}

func (s *mockStep) undo() error {
	s.OutputBuilder.Msg("Uninstalling: " + s.Desc)
	*s.undoCnt++
	return s.Err
}

type MockUbuntuWithError struct {
	BaseK8sInstaller
	errorOnStep int
	curStep     int
	doCnt       int
	undoCnt     int
}

func (u *MockUbuntuWithError) getEmptyStep(desc string, bki *BaseK8sInstaller) Step {
	u.curStep++
	var err error
	if u.curStep == u.errorOnStep {
		err = errors.New("error")
	}
	return &mockStep{
		BaseK8sInstaller: bki,
		Desc:             desc,
		Err:              err,
		doCnt:            &u.doCnt,
		undoCnt:          &u.undoCnt}
}

func (u *MockUbuntuWithError) swapStep(bki *BaseK8sInstaller) Step {
	return u.getEmptyStep("SWAP", bki)
}

func (u *MockUbuntuWithError) firewallStep(bki *BaseK8sInstaller) Step {
	return u.getEmptyStep("FIREWALL", bki)
}

func (u *MockUbuntuWithError) kernelModsLoadStep(bki *BaseK8sInstaller) Step {
	return u.getEmptyStep("KERNEL MODULES", bki)
}

func (u *MockUbuntuWithError) unattendedUpdStep(bki *BaseK8sInstaller) Step {
	return u.getEmptyStep("AUTO OS UPGRADES", bki)
}

func (u *MockUbuntuWithError) osWideCfgUpdateStep(bki *BaseK8sInstaller) Step {
	return u.getEmptyStep("OS CONFIGURATION", bki)
}

func (u *MockUbuntuWithError) criToolsStep(bki *BaseK8sInstaller) Step {
	return u.getEmptyStep("cri-tools.deb", bki)
}

func (u *MockUbuntuWithError) criKubernetesStep(bki *BaseK8sInstaller) Step {
	return u.getEmptyStep("kubernetes-cni.deb", bki)
}

func (u *MockUbuntuWithError) kubectlStep(bki *BaseK8sInstaller) Step {
	return u.getEmptyStep("kubectl.deb", bki)
}

func (u *MockUbuntuWithError) kubeadmStep(bki *BaseK8sInstaller) Step {
	return u.getEmptyStep("kubeadm.deb", bki)
}

func (u *MockUbuntuWithError) kubeletStep(bki *BaseK8sInstaller) Step {
	return u.getEmptyStep("kubelet.deb", bki)
}

func (u *MockUbuntuWithError) containerdStep(bki *BaseK8sInstaller) Step {
	return u.getEmptyStep("CONTAINERD", bki)
}

func (u *MockUbuntuWithError) containerdDaemonStep(bki *BaseK8sInstaller) Step {
	return u.getEmptyStep("CONTAINERD SERVICE", bki)
}
