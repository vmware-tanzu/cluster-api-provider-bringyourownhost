package main

import (
	"flag"

	"github.com/vmware-tanzu/cluster-api-provider-byoh/agent/reconciler"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/agent/registration"
	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/api/v1alpha4"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	hostName  string = "jaime.com"
	namespace string
	scheme    *runtime.Scheme
)

func init() {
	scheme = runtime.NewScheme()
	infrastructurev1alpha4.AddToScheme(scheme)
	corev1.AddToScheme(scheme)
	clusterv1.AddToScheme(scheme)

	flag.StringVar(&namespace, "namespace", "default", "Namespace in the management cluster where you would like to register this host")
}

// TODO - fix logging

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

	registration.HostRegistrar{K8sClient: k8sClient}.Register(hostName, namespace)

	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme:    scheme,
		Namespace: namespace,
	})
	if err != nil {
		klog.Fatal(err, "unable to start manager")
	}

	hostReconciler := reconciler.HostReconciler{Client: k8sClient}
	ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1alpha4.ByoHost{}).
		Complete(hostReconciler)

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		klog.Fatal(err, "problem running manager")
	}
}
