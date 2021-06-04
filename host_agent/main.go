package main

import (
	"context"
	"flag"
	"fmt"

	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/api/v1alpha4"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	"k8s.io/klog/klogr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	hostName  string = "jaime.com"
	namespace string = "default"
	scheme    *runtime.Scheme
)

func init() {
	klog.InitFlags(nil)
	scheme = runtime.NewScheme()
	infrastructurev1alpha4.AddToScheme(scheme)
}

func main() {

	flag.Parse()
	ctrl.SetLogger(klogr.New())

	config := ctrl.GetConfigOrDie()

	k8sClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		fmt.Println(err.Error())
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
		fmt.Println(err.Error())
	}

}
