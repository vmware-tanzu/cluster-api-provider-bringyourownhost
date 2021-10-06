package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/vmware-tanzu/cluster-api-provider-byoh/agent/cloudinit"
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

// labelFlags is a flag that holds a map of label key values.
// One or more key value pairs can be passed using the same flag
// The following example sets labelFlags with two items:
//     -label "key1=value1" -label "key2=value2"
type labelFlags map[string]string

// String implements flag.Value interface
func (l *labelFlags) String() string {
	var result []string
	for key, value := range *l {
		result = append(result, fmt.Sprintf("%s=%s", key, value))
	}
	return strings.Join(result, ",")
}

// nolint: gomnd
// Set implements flag.Value interface
func (l *labelFlags) Set(value string) error {
	// account for comma-separated key-value pairs in a single invocation
	if len(strings.Split(value, ",")) > 1 {
		for _, s := range strings.Split(value, ",") {
			if s == "" {
				continue
			}
			parts := strings.SplitN(s, "=", 2)
			if len(parts) < 2 {
				return fmt.Errorf("invalid argument value. expect key=value, got %s", value)
			}
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			(*l)[k] = v
		}
		return nil
	} else {
		// account for only one key-value pair in a single invocation
		parts := strings.SplitN(value, "=", 2)
		if len(parts) < 2 {
			return fmt.Errorf("invalid argument value. expect key=value, got %s", value)
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		(*l)[k] = v
		return nil
	}
}

var (
	namespace string
	scheme    *runtime.Scheme
	labels    = make(labelFlags)
)

// TODO - fix logging

func main() {
	flag.StringVar(&namespace, "namespace", "default", "Namespace in the management cluster where you would like to register this host")
	flag.Var(&labels, "label", "labels to attach to the ByoHost CR in the form labelname=labelVal for e.g. '--label site=apac --label cores=2'")
	klog.InitFlags(nil)
	flag.Parse()
	scheme = runtime.NewScheme()
	_ = infrastructurev1alpha4.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = clusterv1.AddToScheme(scheme)

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

	registration.LocalHostRegistrar = &registration.HostRegistrar{K8sClient: k8sClient}
	err = registration.LocalHostRegistrar.Register(hostName, namespace, labels)
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
		TemplateParser: cloudinit.TemplateParser{
			Template: registration.HostInfo{
				DefaultNetworkName: registration.LocalHostRegistrar.ByoHostInfo.DefaultNetworkName,
			},
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
