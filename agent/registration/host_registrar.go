package registration

import (
	"context"
	"net"

	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/apis/infrastructure/v1alpha4"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type HostRegistrar struct {
	K8sClient client.Client
}

func (hr HostRegistrar) GetNetworkStatus() []infrastructurev1alpha4.NetworkStatus{
    Network := []infrastructurev1alpha4.NetworkStatus{}
    ifaces, err := net.Interfaces()
    if err != nil {
        return Network
    }

    for _, iface := range ifaces {
        netStatus := infrastructurev1alpha4.NetworkStatus{}

        if iface.Flags & net.FlagUp > 0 {
            netStatus.Connected = true
        }

        netStatus.MACAddr = iface.HardwareAddr.String()
        addrs, err := iface.Addrs()
        if err != nil {
            continue
        }

        netStatus.NetworkName = iface.Name
        for _, addr := range addrs {
            netStatus.IPAddrs = append(netStatus.IPAddrs, addr.String())
        }

        Network = append(Network, netStatus)
    }

    return Network
}

func (hr HostRegistrar) Register(hostName, namespace string) error {
	byoHost := &infrastructurev1alpha4.ByoHost{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ByoHost",
			APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      hostName,
			Namespace: namespace,
		},
		Spec: infrastructurev1alpha4.ByoHostSpec{},
	}

	byoHost.Spec.Network = hr.GetNetworkStatus()
	for _, netStatus := range byoHost.Spec.Network {
		byoHost.Spec.Addresses = append(byoHost.Spec.Addresses, netStatus.IPAddrs...)
	}

	return hr.K8sClient.Create(context.TODO(), byoHost)
}
