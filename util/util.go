// Copyright 2025 Cohesity, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package util implements utilities.
package util

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	errorsutil "k8s.io/apimachinery/pkg/util/errors"
	utilsexec "k8s.io/utils/exec"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrastructurev1beta1 "github.com/cohesity/cluster-api-provider-bringyourownhost/api/infrastructure/v1beta1"
)

// ByoMachineKind is the Kind for ByoMachine
const (
	ByoMachineKind       = "ByoMachine"
	RemoveContainerRetry = 5
)

var (
	// ErrMachineRefNotSet is returned when machineRef is not set in ByoHost status
	ErrMachineRefNotSet = errors.New("machineRef is not set in ByoHost status")

	// ErrMachineRefIncorrectKind is returned when machineRef has incorrect kind
	ErrMachineRefIncorrectKind = errors.New("machineRef has incorrect kind")

	// ErrMachineRefIncorrectGroup is returned when machineRef has incorrect group
	ErrMachineRefIncorrectGroup = errors.New("machineRef has incorrect group")

	// defaultKnownCRISockets holds the set of known CRI endpoints
	defaultKnownCRISockets = []string{
		"unix:///var/run/containerd/containerd.sock",
		"unix:///var/run/crio/crio.sock",
		"unix:///var/run/docker.sock",
	}

	DefaultCRISocket = "unix:///var/run/containerd/containerd.sock"
)

type ContainerRuntime interface {
	Socket() string
	ListContainers() ([]string, error)
	RemoveContainers(ctx context.Context, containers []string) error
}

// CRIRuntime is a struct that interfaces with the CRI
type CRIRuntime struct {
	exec       utilsexec.Interface
	criSocket  string
	crictlPath string
}

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

// detectCRISocketImpl is separated out only for test purposes, DON'T call it directly, use DetectCRISocket instead
func detectCRISocketImpl(isSocket func(string) bool, knownCRISockets []string) (string, error) {
	foundCRISockets := []string{}

	for _, socket := range knownCRISockets {
		if isSocket(socket) {
			foundCRISockets = append(foundCRISockets, socket)
		}
	}

	switch len(foundCRISockets) {
	case 0:
		// Fall back to the default socket if no CRI is detected, we can error out later on if we need it
		return DefaultCRISocket, nil
	case 1:
		// Precisely one CRI found, use that
		return foundCRISockets[0], nil
	default:
		// Multiple CRIs installed?
		return "", fmt.Errorf("Found multiple CRI endpoints on the host. Please define which one do you wish "+
			"to use by setting the 'criSocket' field in the kubeadm configuration file: %s",
			strings.Join(foundCRISockets, ", "))
	}
}

// DetectCRISocket uses a list of known CRI sockets to detect one. If more than one or none is discovered, an error is returned.
func DetectCRISocket() (string, error) {
	return detectCRISocketImpl(isExistingSocket, defaultKnownCRISockets)
}

// isExistingSocket checks if path exists and is domain socket
func isExistingSocket(path string) bool {
	u, err := url.Parse(path)
	if err != nil {
		// should not happen, since we are trying to access known / hardcoded sockets
		return false
	}

	c, err := net.Dial(u.Scheme, u.Path)
	if err != nil {
		return false
	}
	defer c.Close()
	return true
}

// NewContainerRuntime sets up and returns a ContainerRuntime struct
func NewContainerRuntime(execer utilsexec.Interface, criSocket string) (ContainerRuntime, error) {
	const toolName = "crictl"
	crictlPath, err := execer.LookPath(toolName)
	if err != nil {
		return nil, fmt.Errorf("%s is required by the container runtime: %w", toolName, err)
	}
	return &CRIRuntime{execer, criSocket, crictlPath}, nil
}

// ListContainers lists running k8s CRI pods
func (runtime *CRIRuntime) ListContainers() ([]string, error) {
	// Disable debug mode regardless how the crictl is configured so that the debug info won't be
	// iterpreted to the Pod ID.
	args := []string{"-D=false", "pods", "-q"}
	out, err := runtime.crictl(args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("output: %s, error: %w", string(out), err)
	}
	pods := []string{}
	pods = append(pods, strings.Fields(string(out))...)
	return pods, nil
}

// crictl creates a crictl command for the provided args.
func (runtime *CRIRuntime) crictl(args ...string) utilsexec.Cmd {
	cmd := runtime.exec.Command(runtime.crictlPath, append([]string{"-r", runtime.Socket(), "-i", runtime.Socket()}, args...)...)
	cmd.SetEnv(os.Environ())
	return cmd
}

// Socket returns the CRI socket endpoint
func (runtime *CRIRuntime) Socket() string {
	return runtime.criSocket
}

// RemoveContainers removes running k8s pods
func (runtime *CRIRuntime) RemoveContainers(ctx context.Context, containers []string) error {
	logger := ctrl.LoggerFrom(ctx)
	errs := []error{}
	for _, container := range containers {
		var lastErr error
		for i := 0; i < RemoveContainerRetry; i++ {
			logger.V(5).Info("Attempting to remove container", "container", container)
			out, err := runtime.crictl("stopp", container).CombinedOutput()
			if err != nil {
				lastErr = fmt.Errorf("failed to stop running pod %s: output: %s: %w", container, string(out), err)
				continue
			}
			out, err = runtime.crictl("rmp", container).CombinedOutput()
			if err != nil {
				lastErr = fmt.Errorf("failed to remove running container %s: output: %s: %w", container, string(out), err)
				continue
			}
			lastErr = nil
			break
		}

		if lastErr != nil {
			errs = append(errs, lastErr)
		}
	}
	return errorsutil.NewAggregate(errs)
}
