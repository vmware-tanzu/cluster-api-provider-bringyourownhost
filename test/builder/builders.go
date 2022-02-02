// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	infrastructurev1beta1 "github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/apis/infrastructure/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// ByoMachineBuilder holds the variables and objects required to build an infrastructurev1beta1.ByoMachine
type ByoMachineBuilder struct {
	namespace    string
	name         string
	clusterLabel string
	machine      *clusterv1.Machine
	selector     map[string]string
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

// WithLabelSelector adds the passed cluster label to the ByoMachineBuilder
func (b *ByoMachineBuilder) WithLabelSelector(selector map[string]string) *ByoMachineBuilder {
	b.selector = selector
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
	if b.selector != nil {
		byoMachine.Spec.Selector = &metav1.LabelSelector{MatchLabels: b.selector}
	}

	return byoMachine
}

// ByoHostBuilder holds the variables and objects required to build an infrastructurev1beta1.ByoHost
type ByoHostBuilder struct {
	namespace string
	name      string
	labels    map[string]string
}

// ByoHost returns a ByoHostBuilder with the given name and namespace
func ByoHost(namespace, name string) *ByoHostBuilder {
	return &ByoHostBuilder{
		namespace: namespace,
		name:      name,
	}
}

// WithLabels adds the passed labels to the ByoHostBuilder
func (b *ByoHostBuilder) WithLabels(labels map[string]string) *ByoHostBuilder {
	b.labels = labels
	return b
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
	if b.labels != nil {
		byoHost.Labels = b.labels
	}

	return byoHost
}

// MachineBuilder holds the variables and objects required to build a clusterv1.Machine
type MachineBuilder struct {
	namespace           string
	name                string
	cluster             string
	version             string
	bootstrapDataSecret string
}

// ByoClusterBuilder holds the variables and objects required to build an infrastructurev1beta1.ByoCluster
type ByoClusterBuilder struct {
	namespace      string
	name           string
	bundleRegistry string
	bundleTag      string
	cluster        *clusterv1.Cluster
}

// ByoCluster returns a ByoClusterBuilder with the given name and namespace
func ByoCluster(namespace, name string) *ByoClusterBuilder {
	return &ByoClusterBuilder{
		namespace: namespace,
		name:      name,
	}
}

// WithOwnerCluster adds the passed Owner Cluster to the ByoClusterBuilder
func (c *ByoClusterBuilder) WithOwnerCluster(cluster *clusterv1.Cluster) *ByoClusterBuilder {
	c.cluster = cluster
	return c
}

// WithBundleBaseRegistry adds the passed registry value to the ByoClusterBuilder
func (c *ByoClusterBuilder) WithBundleBaseRegistry(registry string) *ByoClusterBuilder {
	c.bundleRegistry = registry
	return c
}

// WithBundleTag adds the passed bundleTag value to the ByoClusterBuilder
func (c *ByoClusterBuilder) WithBundleTag(tag string) *ByoClusterBuilder {
	c.bundleTag = tag
	return c
}

// Build returns a Cluster with the attributes added to the ByoClusterBuilder
func (c *ByoClusterBuilder) Build() *infrastructurev1beta1.ByoCluster {
	cluster := &infrastructurev1beta1.ByoCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ByoCluster",
			APIVersion: clusterv1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.name,
			Namespace: c.namespace,
		},
		Spec: infrastructurev1beta1.ByoClusterSpec{},
	}

	if c.cluster != nil {
		cluster.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
			{
				Kind:       "Cluster",
				Name:       c.cluster.Name,
				APIVersion: clusterv1.GroupVersion.String(),
				UID:        c.cluster.UID,
			},
		}
	}

	if c.bundleRegistry != "" {
		cluster.Spec.BundleLookupBaseRegistry = c.bundleRegistry
	}
	if c.bundleTag != "" {
		cluster.Spec.BundleLookupTag = c.bundleTag
	}

	return cluster
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

// WithBootstrapDataSecret adds the passed bootstrap secret to the MachineBuilder
func (m *MachineBuilder) WithBootstrapDataSecret(secret string) *MachineBuilder {
	m.bootstrapDataSecret = secret
	return m
}

// Build returns a Machine with the attributes added to the MachineBuilder
func (m *MachineBuilder) Build() *clusterv1.Machine {
	machine := &clusterv1.Machine{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Machine",
			APIVersion: clusterv1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: m.name,
			Namespace:    m.namespace,
		},
		Spec: clusterv1.MachineSpec{
			ClusterName: m.cluster,
			Version:     &m.version,
		},
	}
	if m.bootstrapDataSecret != "" {
		machine.Spec.Bootstrap = clusterv1.Bootstrap{
			DataSecretName: &m.bootstrapDataSecret,
		}
	}

	return machine
}

// ClusterBuilder holds the variables and objects required to build a clusterv1.Cluster
type ClusterBuilder struct {
	namespace  string
	name       string
	paused     bool
	byoCluster *infrastructurev1beta1.ByoCluster
}

// Cluster returns a ClusterBuilder with the given name and namespace
func Cluster(namespace, name string) *ClusterBuilder {
	return &ClusterBuilder{
		namespace: namespace,
		name:      name,
	}
}

// WithPausedField adds the passed paused value to the ClusterBuilder
func (c *ClusterBuilder) WithPausedField(paused bool) *ClusterBuilder {
	c.paused = paused
	return c
}

// WithInfrastructureRef adds the passed byoCluster value to the ClusterBuilder
func (c *ClusterBuilder) WithInfrastructureRef(byoCluster *infrastructurev1beta1.ByoCluster) *ClusterBuilder {
	c.byoCluster = byoCluster
	return c
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
	if c.paused {
		cluster.Spec.Paused = c.paused
	}

	if c.byoCluster != nil {
		cluster.Spec.InfrastructureRef = &corev1.ObjectReference{
			Kind:      "ByoCluster",
			Namespace: c.byoCluster.Namespace,
			Name:      c.byoCluster.Name,
			UID:       c.byoCluster.UID,
		}
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
	namespace  string
	name       string
	providerID string
}

// Node returns a NodeBuilder with the given name and namespace
func Node(namespace, name string) *NodeBuilder {
	return &NodeBuilder{
		namespace: namespace,
		name:      name,
	}
}

// WithProviderID adds the passed providerID to the NodeBuilder
func (n *NodeBuilder) WithProviderID(providerID string) *NodeBuilder {
	n.providerID = providerID
	return n
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
		Spec: corev1.NodeSpec{
			ProviderID: n.providerID,
		},
		Status: corev1.NodeStatus{},
	}

	return node
}

// NamespaceBuilder holds the variables and objects required to build a corev1.Namespace
type NamespaceBuilder struct {
	name string
}

// Namespace returns a NamespaceBuilder with the given name
func Namespace(name string) *NamespaceBuilder {
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
