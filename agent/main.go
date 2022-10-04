// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-logr/logr"
	pflag "github.com/spf13/pflag"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/cloudinit"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/reconciler"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/registration"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/version"
	infrastructurev1beta1 "github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/apis/infrastructure/v1beta1"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/feature"
	certv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
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
//nolint: gomnd
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
	flag.Int64Var(&certExpiryDuration, "certExpiryDuration", registration.ExpirationSeconds, "Duration (in seconds) for the expiration of the host certificates")
	flag.Var(&labels, "label", "labels to attach to the ByoHost CR in the form labelname=labelVal for e.g. '--label site=apac --label cores=2'")
	flag.StringVar(&metricsbindaddress, "metricsbindaddress", ":8080", "metricsbindaddress is the TCP address that the controller should bind to for serving prometheus metrics.It can be set to \"0\" to disable the metrics serving")
	flag.StringVar(&downloadpath, "downloadpath", "/var/lib/byoh/bundles", "File System path to keep the downloads")
	flag.BoolVar(&skipInstallation, "skip-installation", false, "If you want to skip installation of the kubernetes component binaries")
	flag.BoolVar(&printVersion, "version", false, "Print the version of the agent")
	flag.StringVar(&bootstrapKubeConfig, "bootstrap-kubeconfig", "", "Provide bootstrap kubeconfig for bootstrap token workflow")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	hiddenFlags := []string{"log-flush-frequency", "alsologtostderr", "log-backtrace-at", "log-dir", "logtostderr", "stderrthreshold", "vmodule", "azure-container-registry-config",
		"log_backtrace_at", "log_dir", "log_file", "log_file_max_size", "add_dir_header", "skip_headers", "skip_log_headers", "one_output", "kubeconfig"}
	for _, hiddenFlag := range hiddenFlags {
		_ = pflag.CommandLine.MarkHidden(hiddenFlag)
	}
	feature.MutableGates.AddFlag(pflag.CommandLine)
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
	namespace           string
	scheme              *runtime.Scheme
	labels              = make(labelFlags)
	metricsbindaddress  string
	downloadpath        string
	skipInstallation    bool
	printVersion        bool
	bootstrapKubeConfig string
	certExpiryDuration  int64
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
	_ = certv1.AddToScheme(scheme)

	logger := klogr.New()
	ctrl.SetLogger(logger)
	hostName, err := os.Hostname()
	if err != nil {
		logger.Error(err, "could not determine hostname")
		return
	}

	_, err = os.Stat(registration.GetBYOHConfigPath())
	// Enable bootstrap flow if --bootstrap-kubeconfig is provided
	// and config doesn't already exists in ~/.byoh/
	if bootstrapKubeConfig != "" && errors.Is(err, os.ErrNotExist) {
		if err = handleBootstrapFlow(logger, hostName); err != nil {
			logger.Error(err, "bootstrap flow failed")
			os.Exit(1)
		}
	}
	// Handle restart flow or if the ~/.byoh/config already exists
	config := getConfig(logger)
	k8sClient := getClient(logger, config)
	registration.LocalHostRegistrar = &registration.HostRegistrar{K8sClient: k8sClient}
	err = registration.LocalHostRegistrar.Register(hostName, namespace, labels)
	if err != nil {
		logger.Error(err, "error registering host %s registration in namespace %s", hostName, namespace)
		return
	}

	// Start certificate rotation goroutine.
	// This is behind a feature flag for now. Set 'CERTIFICATE_ROTATION=true' to enable it.
	if os.Getenv("CERTIFICATE_ROTATION") == "true" {
		go func() {
			err = certificateRotation(logger, hostName, config)
			if err != nil {
				logger.Error(err, "certificate rotation failed")
				return
			}
		}()
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
		logger.Info("skip-installation flag set, skipping installer initialisation")
	}
	hostReconciler := &reconciler.HostReconciler{
		Client:              k8sClient,
		CmdRunner:           cloudinit.CmdRunner{},
		FileWriter:          cloudinit.FileWriter{},
		TemplateParser:      setupTemplateParser(),
		Recorder:            mgr.GetEventRecorderFor("hostagent-controller"),
		SkipK8sInstallation: skipInstallation,
		DownloadPath:        downloadpath,
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

func handleBootstrapFlow(logger logr.Logger, hostName string) error {
	logger.Info("initiated bootstrap kubeconfig flow")
	bootstrapClientConfig, err := registration.LoadRESTClientConfig(bootstrapKubeConfig)
	if err != nil {
		return fmt.Errorf("client config load failed: %v", err)
	}
	byohCSR, err := registration.NewByohCSR(bootstrapClientConfig, logger, certExpiryDuration)
	if err != nil {
		return fmt.Errorf("ByohCSR intialization failed: %v", err)
	}
	err = byohCSR.BootstrapKubeconfig(hostName)
	if err != nil {
		return fmt.Errorf("kubeconfig generation failed: %v", err)
	}
	return nil
}

func certificateRotation(logger logr.Logger, hostName string, config *rest.Config) error {
	var pollDuration = 5 * time.Second
	for {
		block, _ := pem.Decode(config.CertData)
		if block == nil || block.Type != "CERTIFICATE" {
			logger.Info("failed to decode PEM block containing certificate")
			return nil
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			logger.Error(err, "Certifcate parse failed")
			return err
		}

		totalTimeCert := cert.NotAfter.Sub(cert.NotBefore)

		// if less than 20% time left, renew the certs.
		// https://github.com/kubernetes-sigs/cluster-api/blob/main/docs/proposals/20210222-kubelet-authentication.md#kubelet-authenticator-flow
		if time.Now().After(cert.NotAfter.Add(totalTimeCert / -5)) {
			logger.Info("certificate expiration time left is less than 20%, renewing")
			if err = handleBootstrapFlow(logger, hostName); err != nil {
				logger.Error(err, "bootstrap flow failed")
			}
		} else {
			logger.Info("certificate are valid", "will be renewed after", cert.NotAfter.Add(totalTimeCert/-5))
		}

		// Poll after every few seconds
		time.Sleep(pollDuration)
	}
}

func getConfig(logger logr.Logger) *rest.Config {
	config, err := registration.LoadRESTClientConfig(registration.GetBYOHConfigPath())
	if err != nil {
		logger.Error(err, "client config load failed")
		os.Exit(1)
	}
	return config
}

func getClient(logger logr.Logger, config *rest.Config) client.Client {
	k8sClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		logger.Error(err, "k8s client creation failed")
		os.Exit(1)
	}

	return k8sClient
}
