// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	klog "k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientset "k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	byohcontrollers "github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/controllers/infrastructure"

	infrastructurev1beta1 "github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/apis/infrastructure/v1beta1"

	//+kubebuilder:scaffold:imports
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/controllers/remote"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

var (
	scheme               = runtime.NewScheme()
	setupLog             = ctrl.Log.WithName("setup")
	metricsAddr          string
	enableLeaderElection bool
	probeAddr            string
)

func init() {
	klog.InitFlags(nil)
	// clear any discard loggers set by dependecies
	klog.ClearLogger()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(infrastructurev1beta1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme

	utilruntime.Must(clusterv1.AddToScheme(scheme))
	utilruntime.Must(admissionv1beta1.AddToScheme(scheme))
}

func setFlags() {
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.Parse()
}

// TODO:
// main() will have lots of 'if', '&&' and '||' which will
// increase its cyclometric complexity. Ignoring it for now.

//nolint: funlen, gocyclo
func main() {
	setFlags()
	ctrl.SetLogger(klogr.New())

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "controller-leader-election-caph",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	remoteLogger := ctrl.Log.WithName("remote").WithName("ClusterCacheTracker")
	options := remote.ClusterCacheTrackerOptions{Log: &remoteLogger}
	tracker, err := remote.NewClusterCacheTracker(mgr, options)
	if err != nil {
		setupLog.Error(err, "unable to create cluster cache tracker")
		os.Exit(1)
	}

	if err = (&remote.ClusterCacheReconciler{
		Client:  mgr.GetClient(),
		Log:     ctrl.Log.WithName("remote").WithName("ClusterCacheReconciler"),
		Tracker: tracker,
	}).SetupWithManager(context.TODO(), mgr, concurrency(0)); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ClusterCacheReconciler")
		os.Exit(1)
	}

	if err = (&byohcontrollers.ByoMachineReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Tracker:  tracker,
		Recorder: mgr.GetEventRecorderFor("byomachine-controller"),
	}).SetupWithManager(context.TODO(), mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ByoMachine")
		os.Exit(1)
	}
	if err = (&byohcontrollers.ByoHostReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ByoHost")
		os.Exit(1)
	}
	if err = (&byohcontrollers.ByoMachineTemplateReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ByoMachineTemplate")
		os.Exit(1)
	}
	if err = (&byohcontrollers.ByoClusterReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ByoCluster")
		os.Exit(1)
	}

	// Set 'MANUAL_CSR_APPROVAL=enable' to disable ByoAdmission controller. Now CSRs should be approved manually.
	if os.Getenv("MANUAL_CSR_APPROVAL") != "enable" {
		if err = (&byohcontrollers.ByoAdmissionReconciler{
			ClientSet: clientset.NewForConfigOrDie(ctrl.GetConfigOrDie()),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "ByoAdmission")
			os.Exit(1)
		}
	}
	if err = (&byohcontrollers.K8sInstallerConfigReconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme()}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "K8sInstallerConfig")
		os.Exit(1)
	}

	mgr.GetWebhookServer().Register("/validate-infrastructure-cluster-x-k8s-io-v1beta1-byohost", &webhook.Admission{Handler: &infrastructurev1beta1.ByoHostValidator{}})

	if err = (&byohcontrollers.BootstrapKubeconfigReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "BootstrapKubeconfig")
		os.Exit(1)
	}
	if err = (&infrastructurev1beta1.BootstrapKubeconfig{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "BootstrapKubeconfig")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func concurrency(c int) controller.Options {
	return controller.Options{MaxConcurrentReconciles: c}
}
