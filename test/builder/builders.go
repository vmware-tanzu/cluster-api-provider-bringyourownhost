// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	infrastructurev1beta1 "github.com/vmware-tanzu/cluster-api-provider-byoh/apis/infrastructure/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// ByoMachineBuider holds the variables and objects required to build a infrastructurev1beta1.ByoMachine
type ByoMachineBuilder struct {
	namespace    string
	name         string
	clusterLabel string
	machine      *clusterv1.Machine
}

// ByoMachine returns a ByoMachineBuilder with the given name and namespace
func ByoMachine(namespace, name string) *ByoMachineBuilder {
	return &ByoMachineBuilder{
		namespace: namespace,
		name:      name,
	}
}

// WithOwnerMachine adds the passed Owner Machine to the ByoMachineBuilder
func (b *ByoMachineBuilder) WithOwnerMachine(machine *clusterv1.Machine) *ByoMachineBuilder {
	b.machine = machine
	return b
}

// WithClusterLabel adds the passed cluster label to the ByoMachineBuilder
func (b *ByoMachineBuilder) WithClusterLabel(clusterName string) *ByoMachineBuilder {
	b.clusterLabel = clusterName
	return b
}

// Build returns a ByoMachine with the attributes added to the ByoMachineBuilder
func (b *ByoMachineBuilder) Build() *infrastructurev1beta1.ByoMachine {
	byoMachine := &infrastructurev1beta1.ByoMachine{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ByoMachine",
			APIVersion: infrastructurev1beta1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: b.name,
			Namespace:    b.namespace,
		},
		Spec: infrastructurev1beta1.ByoMachineSpec{},
	}
	if b.machine != nil {
		byoMachine.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
			{
				Kind:       "Machine",
				Name:       b.machine.Name,
				APIVersion: clusterv1.GroupVersion.String(),
				UID:        b.machine.UID,
			},
		}
	}
	if b.clusterLabel != "" {
		byoMachine.ObjectMeta.Labels = map[string]string{
			clusterv1.ClusterLabelName: b.clusterLabel,
		}
	}

	return byoMachine
}

// ByoHostBuilder holds the variables and objects required to build a infrastructurev1beta1.ByoHost
type ByoHostBuilder struct {
	namespace string
	name      string
}

// ByoHost returns a ByoHostBuilder with the given name and namespace
func ByoHost(namespace, name string) *ByoHostBuilder {
	return &ByoHostBuilder{
		namespace: namespace,
		name:      name,
	}
}

// Build returns a ByoHost with the attributes added to the ByoHostBuilder
func (b *ByoHostBuilder) Build() *infrastructurev1beta1.ByoHost {
	byoHost := &infrastructurev1beta1.ByoHost{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ByoHost",
			APIVersion: infrastructurev1beta1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: b.name,
			Namespace:    b.namespace,
		},
		Spec: infrastructurev1beta1.ByoHostSpec{},
	}

	return byoHost
}

// MachineBuilder holds the variables and objects required to build a clusterv1.Machine
type MachineBuilder struct {
	namespace string
	name      string
	cluster   string
	version   string
}

// Machine returns a MachineBuilder with the given name and namespace
func Machine(namespace, name string) *MachineBuilder {
	return &MachineBuilder{
		namespace: namespace,
		name:      name,
	}
}

// WithClusterName adds the passed Cluster to the MachineBuilder
func (m *MachineBuilder) WithClusterName(cluster string) *MachineBuilder {
	m.cluster = cluster
	return m
}

// WithClusterVersion adds the passed cluster version to the MachineBuilder
func (m *MachineBuilder) WithClusterVersion(version string) *MachineBuilder {
	m.version = version
	return m
}

// Build returns a Machine with the attributes added to the MachineBuilder
func (b *MachineBuilder) Build() *clusterv1.Machine {
	machine := &clusterv1.Machine{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Machine",
			APIVersion: clusterv1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: b.name,
			Namespace:    b.namespace,
		},
		Spec: clusterv1.MachineSpec{
			ClusterName: b.cluster,
			Version:     &b.version,
		},
	}

	return machine
}

// ClusterBuilder holds the variables and objects required to build a clusterv1.Cluster
type ClusterBuilder struct {
	namespace string
	name      string
}

// Cluster returns a ClusterBuilder with the given name and namespace
func Cluster(namespace, name string) *ClusterBuilder {
	return &ClusterBuilder{
		namespace: namespace,
		name:      name,
	}
}

// Build returns a Cluster with the attributes added to the ClusterBuilder
func (c *ClusterBuilder) Build() *clusterv1.Cluster {
	cluster := &clusterv1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: clusterv1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.name,
			Namespace: c.namespace,
		},
		Spec: clusterv1.ClusterSpec{},
	}

	return cluster
}

// SecretBuilder holds the variables and objects required to build a corev1.Secret
type SecretBuilder struct {
	namespace string
	name      string
	data      map[string][]byte
}

// Secret returns a SecretBuilder with the given name and namespace
func Secret(namespace, name string) *SecretBuilder {
	return &SecretBuilder{
		namespace: namespace,
		name:      name,
	}
}

// WithData adds the passed data to the SecretBuilder
func (s *SecretBuilder) WithData(value string) *SecretBuilder {
	s.data = map[string][]byte{
		"value": []byte(value),
	}
	return s
}

// Build returns a Secret with the attributes added to the SecretBuilder
func (s *SecretBuilder) Build() *corev1.Secret {
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.name,
			Namespace: s.namespace,
		},
		Data: s.data,
	}

	return secret
}

// NodeBuilder holds the variables and objects required to build a corev1.Node
type NodeBuilder struct {
	namespace string
	name      string
}

// Node returns a NodeBuilder with the given name and namespace
func Node(namespace, name string) *NodeBuilder {
	return &NodeBuilder{
		namespace: namespace,
		name:      name,
	}
}

// Build returns a Node with the attributes added to the NodeBuilder
func (n *NodeBuilder) Build() *corev1.Node {
	node := &corev1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      n.name,
			Namespace: n.namespace,
		},
		Spec:   corev1.NodeSpec{},
		Status: corev1.NodeStatus{},
	}

	return node
}

// NamespaceBuilder holds the variables and objects required to build a corev1.Namespace
type NamespaceBuilder struct {
	name string
}

// Namespace returns a NamespaceBuilder with the given name
func Namespace(namespace, name string) *NamespaceBuilder {
	return &NamespaceBuilder{
		name: name,
	}
}

// Build returns a Namespace with the attributes added to the NamespaceBuilder
func (n *NamespaceBuilder) Build() *corev1.Namespace {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{GenerateName: n.name},
	}

	return namespace
}
