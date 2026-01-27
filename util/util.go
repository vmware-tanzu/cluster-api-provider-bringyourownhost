// Copyright 2025 Cohesity, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package util implements utilities.
package util

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrastructurev1beta1 "github.com/cohesity/cluster-api-provider-bringyourownhost/api/infrastructure/v1beta1"
)

const (
	// ByoMachineKind is the Kind for ByoMachine
	ByoMachineKind = "ByoMachine"
)

var (
	// ErrMachineRefNotSet is returned when machineRef is not set in ByoHost status
	ErrMachineRefNotSet = errors.New("machineRef is not set in ByoHost status")

	// ErrMachineRefIncorrectKind is returned when machineRef has incorrect kind
	ErrMachineRefIncorrectKind = errors.New("machineRef has incorrect kind")

	// ErrMachineRefIncorrectGroup is returned when machineRef has incorrect group
	ErrMachineRefIncorrectGroup = errors.New("machineRef has incorrect group")
)

// GetOwnerByoMachine returns the ByoMachine object owning the current resource.
func GetOwnerByoMachine(ctx context.Context, c client.Client, obj *metav1.ObjectMeta) (*infrastructurev1beta1.ByoMachine, error) {
	for _, ref := range obj.OwnerReferences {
		gv, err := schema.ParseGroupVersion(ref.APIVersion)
		if err != nil {
			return nil, err
		}
		if ref.Kind == "ByoMachine" && gv.Group == infrastructurev1beta1.GroupVersion.Group {
			return GetByoMachineByName(ctx, c, obj.Namespace, ref.Name)
		}
	}
	return nil, nil
}

// GetByoMachineByName finds and return a ByoMachine object using the specified params.
func GetByoMachineByName(ctx context.Context, c client.Client, namespace, name string) (*infrastructurev1beta1.ByoMachine, error) {
	m := &infrastructurev1beta1.ByoMachine{}
	key := client.ObjectKey{Name: name, Namespace: namespace}
	if err := c.Get(ctx, key, m); err != nil {
		return nil, err
	}
	return m, nil
}

func GetNodeForByoHost(ctx context.Context, c client.Client, byoHost *infrastructurev1beta1.ByoHost) (*corev1.Node, error) {
	node := &corev1.Node{}
	key := client.ObjectKey{Name: byoHost.Name, Namespace: byoHost.Namespace}
	if err := c.Get(ctx, key, node); err != nil {
		return nil, fmt.Errorf("failed to get node for ByoHost: %w", err)
	}
	return node, nil
}

// GetByoMachineForHost validates and fetches the ByoMachine referenced in the ByoHost status.
// It validates that the MachineRef has the correct Kind and Group before fetching.
func GetByoMachineForHost(ctx context.Context, c client.Client, byoHost *infrastructurev1beta1.ByoHost) (*infrastructurev1beta1.ByoMachine, error) {
	// Check if the byoHost has a machineRef set
	if byoHost.Status.MachineRef == nil {
		return nil, ErrMachineRefNotSet
	}

	// Validate MachineRef Kind
	if byoHost.Status.MachineRef.Kind != ByoMachineKind {
		return nil, fmt.Errorf("%w: expected=%s, got=%s", ErrMachineRefIncorrectKind, ByoMachineKind, byoHost.Status.MachineRef.Kind)
	}

	// Validate MachineRef Group
	gv, err := schema.ParseGroupVersion(byoHost.Status.MachineRef.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to parse machineRef APIVersion: %w", err)
	}

	if gv.Group != infrastructurev1beta1.GroupVersion.Group {
		return nil, fmt.Errorf("%w: expected=%s, got=%s", ErrMachineRefIncorrectGroup, infrastructurev1beta1.GroupVersion.Group, gv.Group)
	}

	// Get the ByoMachine object from the machineRef
	byoMachine := &infrastructurev1beta1.ByoMachine{}
	byoMachineKey := client.ObjectKey{
		Namespace: byoHost.Status.MachineRef.Namespace,
		Name:      byoHost.Status.MachineRef.Name,
	}

	if err := c.Get(ctx, byoMachineKey, byoMachine); err != nil {
		return nil, fmt.Errorf("failed to get ByoMachine from machineRef: %w", err)
	}

	return byoMachine, nil
}
