// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package common

import (
	infrastructurev1beta1 "github.com/vmware-tanzu/cluster-api-provider-byoh/apis/infrastructure/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

func NewByoMachine(byoMachineName, byoMachineNamespace, clusterName string, machine *clusterv1.Machine) *infrastructurev1beta1.ByoMachine {
	byoMachine := &infrastructurev1beta1.ByoMachine{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ByoMachine",
			APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: byoMachineName,
			Namespace:    byoMachineNamespace,
		},
		Spec: infrastructurev1beta1.ByoMachineSpec{},
	}

	if machine != nil {
		byoMachine.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
			{
				Kind:       "Machine",
				Name:       machine.Name,
				APIVersion: "cluster.x-k8s.io/v1beta1",
				UID:        machine.UID,
			},
		}
	}

	if len(clusterName) > 0 {
		byoMachine.ObjectMeta.Labels = map[string]string{
			clusterv1.ClusterLabelName: clusterName,
		}
	}
	return byoMachine
}

func NewMachine(machineName, namespace, clusterName string) *clusterv1.Machine {
	testClusterVersion := "1.22"
	machine := &clusterv1.Machine{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Machine",
			APIVersion: "cluster.x-k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: machineName,
			Namespace:    namespace,
		},
		Spec: clusterv1.MachineSpec{
			ClusterName: clusterName,
			Version:     &testClusterVersion,
		},
	}
	return machine
}

func NewByoHost(byoHostName, byoHostNamespace string) *infrastructurev1beta1.ByoHost {
	byoHost := &infrastructurev1beta1.ByoHost{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ByoHost",
			APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: byoHostName,
			Namespace:    byoHostNamespace,
		},
		Spec: infrastructurev1beta1.ByoHostSpec{},
	}
	return byoHost
}

func NewNode(nodeName, namespace string) *corev1.Node {
	node := &corev1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      nodeName,
			Namespace: namespace,
		},
		Spec:   corev1.NodeSpec{},
		Status: corev1.NodeStatus{},
	}
	return node
}

func NewCluster(clusterName, namespace string) *clusterv1.Cluster {
	cluster := &clusterv1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "cluster.x-k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: namespace,
		},
		Spec: clusterv1.ClusterSpec{},
	}
	return cluster
}

func NewNamespace(namespace string) *corev1.Namespace {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{GenerateName: namespace},
	}
	return ns
}

func NewSecret(bootstrapSecretName, stringDataValue, namespace string) *corev1.Secret {
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      bootstrapSecretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"value": []byte(stringDataValue),
		},
		Type: "cluster.x-k8s.io/secret",
	}
	return secret
}
