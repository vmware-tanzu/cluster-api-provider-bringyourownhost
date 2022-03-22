// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package algo

import "errors"

type mockStep struct {
	Step
	Desc string
	Err  error
	*BaseK8sInstaller
	*MockUbuntuWithError
}

func (s *mockStep) do() error {
	s.doSteps = append(s.doSteps, s.Desc)
	return s.Err
}

func (s *mockStep) undo() error {
	s.undoSteps = append(s.undoSteps, s.Desc)
	return s.Err
}

// MockUbuntuWithError is a mock implementation of BaseK8sInstaller that returns an error on the steps
type MockUbuntuWithError struct {
	BaseK8sInstaller
	errorOnStep int
	curStep     int
	doSteps     []string
	undoSteps   []string
}

func (u *MockUbuntuWithError) getEmptyStep(desc string, bki *BaseK8sInstaller) Step {
	u.curStep++
	var err error
	if u.curStep == u.errorOnStep {
		err = errors.New("error")
	}
	return &mockStep{
		BaseK8sInstaller:    bki,
		Desc:                desc,
		Err:                 err,
		MockUbuntuWithError: u}
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
