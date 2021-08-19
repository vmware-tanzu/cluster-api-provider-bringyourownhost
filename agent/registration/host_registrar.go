package registration

import (
	"context"

	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/apis/infrastructure/v1alpha4"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type HostRegistrar struct {
	K8sClient client.Client
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

	return hr.K8sClient.Create(context.TODO(), byoHost)
}
