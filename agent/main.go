package main

import (
	"context"
	"flag"
	"os"

	"github.com/vmware-tanzu/cluster-api-provider-byoh/agent/cloudinit"
	agentConfig "github.com/vmware-tanzu/cluster-api-provider-byoh/agent/config"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/agent/reconciler"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/agent/registration"
	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/apis/infrastructure/v1alpha4"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	"k8s.io/klog/klogr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	namespace string
	scheme    *runtime.Scheme
)

func init() {
	klog.InitFlags(nil)
	scheme = runtime.NewScheme()
	_ = infrastructurev1alpha4.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = clusterv1.AddToScheme(scheme)

	flag.StringVar(&namespace, "namespace", "default", "Namespace in the management cluster where you would like to register this host")
}

// TODO - fix logging

func main() {
	flag.Parse()
	ctrl.SetLogger(klogr.New())
	config, err := ctrl.GetConfig()
	if err != nil {
		klog.Errorf("error getting kubeconfig, err=%v", err)
		return
	}

	k8sClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		klog.Errorf("error creating a new k8s client, err=%v", err)
		return
	}

	hostName, err := os.Hostname()
	if err != nil {
		klog.Errorf("couldn't determine hostname, err=%v", err)
		return
	}

	hostRegistrar := &registration.HostRegistrar{K8sClient: k8sClient}
	err = hostRegistrar.Register(hostName, namespace)
	if err != nil {
		klog.Errorf("error registering host %s registration in namespace %s, err=%v", hostName, namespace, err)
		return
	}

	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme:    scheme,
		Namespace: namespace,
		// this enables filtered watch of ByoHost based on the host name
		// only ByoHost running for this host will be cached
		NewCache: cache.BuilderWithOptions(cache.Options{
			SelectorsByObject: cache.SelectorsByObject{
				&infrastructurev1alpha4.ByoHost{}: {
					Field: fields.SelectorFromSet(fields.Set{"metadata.name": hostName}),
				},
			},
		},
		),
	})
	if err != nil {
		klog.Errorf("unable to start manager, err=%v", err)
		return
	}

	hostReconciler := &reconciler.HostReconciler{
		Client:     k8sClient,
		CmdRunner:  cloudinit.CmdRunner{},
		FileWriter: cloudinit.FileWriter{},
		HostInfo: agentConfig.HostInfo{
			DefaultNetworkName: hostRegistrar.HostInfo.DefaultNetworkName,
		},
	}
	if err = hostReconciler.SetupWithManager(context.TODO(), mgr); err != nil {
		klog.Errorf("unable to create controller, err=%v", err)
		return
	}

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		klog.Errorf("problem running manager, err=%v", err)
		return
	}
}
