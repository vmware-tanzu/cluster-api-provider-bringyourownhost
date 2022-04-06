// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package registration

import (
	"context"
	"net"

	"github.com/jackpal/gateway"
	infrastructurev1beta1 "github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/apis/infrastructure/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	klog "k8s.io/klog/v2"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// LocalHostRegistrar is a HostRegistrar that registers the local host.
	LocalHostRegistrar *HostRegistrar
)

// HostInfo contains information about the host network interface.
type HostInfo struct {
	DefaultNetworkInterfaceName string
}

// HostRegistrar used to register a host.
type HostRegistrar struct {
	K8sClient   client.Client
	ByoHostInfo HostInfo
}

// Register is called on agent startup
// This function registers the byohost as available capacity in the management cluster
// If the CR is already present, we consider this to be a restart / reboot of the agent process
func (hr *HostRegistrar) Register(hostName, namespace string, hostLabels map[string]string) error {
	klog.Info("Registering ByoHost")
	ctx := context.TODO()
	byoHost := &infrastructurev1beta1.ByoHost{}
	err := hr.K8sClient.Get(ctx, types.NamespacedName{Name: hostName, Namespace: namespace}, byoHost)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			klog.Errorf("error getting host %s in namespace %s, err=%v", hostName, namespace, err)
			return err
		}
		byoHost = &infrastructurev1beta1.ByoHost{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ByoHost",
				APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      hostName,
				Namespace: namespace,
				Labels:    hostLabels,
			},
			Spec:   infrastructurev1beta1.ByoHostSpec{},
			Status: infrastructurev1beta1.ByoHostStatus{},
		}
		err = hr.K8sClient.Create(ctx, byoHost)
		if err != nil {
			klog.Errorf("error creating host %s in namespace %s, err=%v", hostName, namespace, err)
			return err
		}
	}

	// run it at startup or reboot
	return hr.UpdateNetwork(ctx, byoHost)
}

// UpdateNetwork updates the network interface status for the host
func (hr *HostRegistrar) UpdateNetwork(ctx context.Context, byoHost *infrastructurev1beta1.ByoHost) error {
	klog.Info("Add Network Info")
	helper, err := patch.NewHelper(byoHost, hr.K8sClient)
	if err != nil {
		return err
	}

	byoHost.Status.Network = hr.GetNetworkStatus()

	return helper.Patch(ctx, byoHost)
}

// GetNetworkStatus returns the network interface(s) status for the host
func (hr *HostRegistrar) GetNetworkStatus() []infrastructurev1beta1.NetworkStatus {
	Network := make([]infrastructurev1beta1.NetworkStatus, 0)

	defaultIP, err := gateway.DiscoverInterface()
	if err != nil {
		return Network
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return Network
	}

	for _, iface := range ifaces {
		netStatus := infrastructurev1beta1.NetworkStatus{}

		if iface.Flags&net.FlagUp > 0 {
			netStatus.Connected = true
		}

		netStatus.MACAddr = iface.HardwareAddr.String()
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		netStatus.NetworkInterfaceName = iface.Name
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip.String() == defaultIP.String() {
				netStatus.IsDefault = true
				hr.ByoHostInfo.DefaultNetworkInterfaceName = netStatus.NetworkInterfaceName
			}
			netStatus.IPAddrs = append(netStatus.IPAddrs, addr.String())
		}
		Network = append(Network, netStatus)
	}
	return Network
}
