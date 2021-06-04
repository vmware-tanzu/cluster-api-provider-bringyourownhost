package main

import (
	"context"
	"flag"

	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/api/v1alpha4"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	hostName  string = "jaime.com"
	namespace string = "default"
	scheme    *runtime.Scheme
)

func init() {
	scheme = runtime.NewScheme()
	infrastructurev1alpha4.AddToScheme(scheme)

}

func main() {
	flag.Parse()

	config, err := ctrl.GetConfig()
	if err != nil {
		klog.Fatal(err)
	}

	k8sClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		klog.Fatal(err)
	}

	ByoHost := &infrastructurev1alpha4.ByoHost{
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

	err = k8sClient.Create(context.Background(), ByoHost)

	if err != nil {
		klog.Fatal(err)
	}

}
