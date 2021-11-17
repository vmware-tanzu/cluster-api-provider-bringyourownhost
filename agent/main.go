// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/cloudinit"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/reconciler"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/registration"
	infrastructurev1beta1 "github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/apis/infrastructure/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	"k8s.io/klog/klogr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
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
	namespace          string
	scheme             *runtime.Scheme
	labels             = make(labelFlags)
	metricsbindaddress string
	downloadpath       string
	skipInstallation   bool
)

// TODO - fix logging

func main() {
	flag.StringVar(&namespace, "namespace", "default", "Namespace in the management cluster where you would like to register this host")
	flag.Var(&labels, "label", "labels to attach to the ByoHost CR in the form labelname=labelVal for e.g. '--label site=apac --label cores=2'")
	flag.StringVar(&metricsbindaddress, "metricsbindaddress", ":8080", "metricsbindaddress is the TCP address that the controller should bind to for serving prometheus metrics.It can be set to \"0\" to disable the metrics serving")
	flag.StringVar(&downloadpath, "downloadpath", "/var/lib/byoh/bundles", "File System path to keep the downloads")
	flag.BoolVar(&skipInstallation, "skip-installation", false, "If you want to skip installation of the kubernetes component binaries")
	klog.InitFlags(nil)
	flag.Parse()
	scheme = runtime.NewScheme()
	_ = infrastructurev1beta1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = clusterv1.AddToScheme(scheme)

	logger := klogr.New()
	ctrl.SetLogger(logger)
	config, err := ctrl.GetConfig()
	if err != nil {
		logger.Error(err, "error getting kubeconfig")
		return
	}

	k8sClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		logger.Error(err, "error creating a new k8s client")
		return
	}

	hostName, err := os.Hostname()
	if err != nil {
		logger.Error(err, "could not determine hostname")
		return
	}

	registration.LocalHostRegistrar = &registration.HostRegistrar{K8sClient: k8sClient}
	err = registration.LocalHostRegistrar.Register(hostName, namespace, labels)
	if err != nil {
		logger.Error(err, "error registering host %s registration in namespace %s", hostName, namespace)
		return
	}

	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme:    scheme,
		Namespace: namespace,
		// this enables filtered watch of ByoHost based on the host name
		// only ByoHost running for this host will be cached
		NewCache: cache.BuilderWithOptions(cache.Options{
			SelectorsByObject: cache.SelectorsByObject{
				&infrastructurev1beta1.ByoHost{}: {
					Field: fields.SelectorFromSet(fields.Set{"metadata.name": hostName}),
				},
			},
		},
		),
		MetricsBindAddress: metricsbindaddress,
	})
	if err != nil {
		logger.Error(err, "unable to start manager")
		return
	}

	hostReconciler := &reconciler.HostReconciler{
		Client:     k8sClient,
		CmdRunner:  cloudinit.CmdRunner{},
		FileWriter: cloudinit.FileWriter{},
		TemplateParser: cloudinit.TemplateParser{
			Template: registration.HostInfo{
				DefaultNetworkInterfaceName: registration.LocalHostRegistrar.ByoHostInfo.DefaultNetworkInterfaceName,
			},
		},
		SkipInstallation: skipInstallation,
		DownloadPath:     downloadpath,
		Recorder:         mgr.GetEventRecorderFor("hostagent-controller"),
	}
	if err = hostReconciler.SetupWithManager(context.TODO(), mgr); err != nil {
		logger.Error(err, "unable to create controller")
		return
	}

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		logger.Error(err, "problem running manager")
		return
	}
}
