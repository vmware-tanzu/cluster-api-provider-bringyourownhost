// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/go-logr/logr"
	pflag "github.com/spf13/pflag"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/cloudinit"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/installer"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/reconciler"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/registration"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/version"
	infrastructurev1beta1 "github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/apis/infrastructure/v1beta1"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/feature"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	klog "k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
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

// Set implements flag.Value interface
// nolint: gomnd
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

func setupflags() {
	klog.InitFlags(nil)
	// clear any discard loggers set by dependecies
	klog.ClearLogger()

	flag.StringVar(&namespace, "namespace", "default", "Namespace in the management cluster where you would like to register this host")
	flag.Var(&labels, "label", "labels to attach to the ByoHost CR in the form labelname=labelVal for e.g. '--label site=apac --label cores=2'")
	flag.StringVar(&metricsbindaddress, "metricsbindaddress", ":8080", "metricsbindaddress is the TCP address that the controller should bind to for serving prometheus metrics.It can be set to \"0\" to disable the metrics serving")
	flag.StringVar(&downloadpath, "downloadpath", "/var/lib/byoh/bundles", "File System path to keep the downloads")
	flag.BoolVar(&skipInstallation, "skip-installation", false, "If you want to skip installation of the kubernetes component binaries")
	flag.BoolVar(&printVersion, "version", false, "Print the version of the agent")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	hiddenFlags := []string{"log-flush-frequency", "alsologtostderr", "log-backtrace-at", "log-dir", "logtostderr", "stderrthreshold", "vmodule", "azure-container-registry-config",
		"log_backtrace_at", "log_dir", "log_file", "log_file_max_size", "add_dir_header", "skip_headers", "skip_log_headers", "one_output"}
	for _, hiddenFlag := range hiddenFlags {
		_ = pflag.CommandLine.MarkHidden(hiddenFlag)
	}
	feature.MutableGates.AddFlag(pflag.CommandLine)
}

func handleHostRegistration(k8sClient client.Client, hostName string, logger logr.Logger) (err error) {
	registration.LocalHostRegistrar = &registration.HostRegistrar{K8sClient: k8sClient}
	if feature.Gates.Enabled(feature.SecureAccess) {
		logger.Info("secure access enabled, waiting for host to be registered by ByoAdmission Controller")
	} else {
		err := registration.LocalHostRegistrar.Register(hostName, namespace, labels)
		return err
	}
	return nil
}

func setupTemplateParser() *cloudinit.TemplateParser {
	var templateParser *cloudinit.TemplateParser
	if registration.LocalHostRegistrar.ByoHostInfo.DefaultNetworkInterfaceName == "" {
		templateParser = nil
	} else {
		templateParser = &cloudinit.TemplateParser{
			Template: registration.HostInfo{
				DefaultNetworkInterfaceName: registration.LocalHostRegistrar.ByoHostInfo.DefaultNetworkInterfaceName,
			},
		}
	}

	return templateParser
}

var (
	namespace          string
	scheme             *runtime.Scheme
	labels             = make(labelFlags)
	metricsbindaddress string
	downloadpath       string
	skipInstallation   bool
	printVersion       bool
	k8sInstaller       reconciler.IK8sInstaller
)

// TODO - fix logging

func main() {
	setupflags()
	pflag.Parse()

	if printVersion {
		info := version.Get()
		fmt.Printf("byoh-hostagent version: %#v\n", info)
		return
	}
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

	err = handleHostRegistration(k8sClient, hostName, logger)
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

	if skipInstallation {
		k8sInstaller = nil
		logger.Info("skip-installation flag set, skipping installer initialisation")
	} else {
		// increasing installer log level to 1, so that it wont be logged by default
		k8sInstaller, err = installer.New(downloadpath, installer.BundleTypeK8s, logger.V(1))
		if err != nil {
			logger.Error(err, "failed to instantiate installer")
		}
	}

	hostReconciler := &reconciler.HostReconciler{
		Client:         k8sClient,
		CmdRunner:      cloudinit.CmdRunner{},
		FileWriter:     cloudinit.FileWriter{},
		TemplateParser: setupTemplateParser(),
		Recorder:       mgr.GetEventRecorderFor("hostagent-controller"),
		K8sInstaller:   k8sInstaller,
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
