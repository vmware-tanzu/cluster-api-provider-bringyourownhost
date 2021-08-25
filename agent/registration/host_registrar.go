package registration

import (
	"context"
	"net"

	"github.com/jackpal/gateway"
	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/apis/infrastructure/v1alpha4"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type HostRegistrar struct {
	K8sClient client.Client
}

func (hr HostRegistrar) Register(hostName, namespace string) error {
	ctx := context.TODO()
	byoHost := &infrastructurev1alpha4.ByoHost{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ByoHost",
			APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      hostName,
			Namespace: namespace,
		},
		Spec:   infrastructurev1alpha4.ByoHostSpec{},
		Status: infrastructurev1alpha4.ByoHostStatus{},
	}

	err := hr.K8sClient.Create(ctx, byoHost)
	if err != nil {
		return err
	}

	// run it at startup
	return hr.UpdateNetwork(ctx, byoHost)
}

func (hr HostRegistrar) UpdateNetwork(ctx context.Context, byoHost *infrastructurev1alpha4.ByoHost) error {
	helper, err := patch.NewHelper(byoHost, hr.K8sClient)
	if err != nil {
		return err
	}

	byoHost.Status.Network = hr.GetNetworkStatus()

	return helper.Patch(ctx, byoHost)
}

func (hr HostRegistrar) GetNetworkStatus() []infrastructurev1alpha4.NetworkStatus {
	Network := []infrastructurev1alpha4.NetworkStatus{}

	defaultIP, err := gateway.DiscoverInterface()
	if err != nil {
		return Network
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return Network
	}

	for _, iface := range ifaces {
		netStatus := infrastructurev1alpha4.NetworkStatus{}

		if iface.Flags&net.FlagUp > 0 {
			netStatus.Connected = true
		}

		netStatus.MACAddr = iface.HardwareAddr.String()
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		netStatus.NetworkName = iface.Name
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
			}
			netStatus.IPAddrs = append(netStatus.IPAddrs, addr.String())
		}
		Network = append(Network, netStatus)
	}
	return Network
}
