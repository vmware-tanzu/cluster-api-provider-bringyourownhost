/*
Copyright the Cluster API Provider BYOH contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	infrav1 "vmware-tanzu/cluster-api-provider-byoh/api/v1alpha3"
	"vmware-tanzu/cluster-api-provider-byoh/controllers"

	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog"
	"k8s.io/klog/klogr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/cmd/version"
	"sigs.k8s.io/cluster-api/util"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")

	// flags
	metricsAddr                 string
	enableLeaderElection        bool
	leaderElectionLeaseDuration time.Duration
	leaderElectionRenewDeadline time.Duration
	leaderElectionRetryPeriod   time.Duration
	watchNamespace              string
	profilerAddress             string
	machineConcurrency          int
	syncPeriod                  time.Duration
	webhookPort                 int
	healthAddr                  string
)

func init() {
	klog.InitFlags(nil)

	_ = clientgoscheme.AddToScheme(scheme)
	_ = clusterv1.AddToScheme(scheme)
	_ = infrav1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

// InitFlags initializes the flags.
func InitFlags(fs *pflag.FlagSet) {
	fs.StringVar(&metricsAddr, "metrics-addr", ":8080",
		"The address the metric endpoint binds to.")

	fs.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")

	fs.DurationVar(&leaderElectionLeaseDuration, "leader-election-lease-duration", 15*time.Second,
		"Interval at which non-leader candidates will wait to force acquire leadership (duration string)")

	fs.DurationVar(&leaderElectionRenewDeadline, "leader-election-renew-deadline", 10*time.Second,
		"Duration that the leading controller manager will retry refreshing leadership before giving up (duration string)")

	fs.DurationVar(&leaderElectionRetryPeriod, "leader-election-retry-period", 2*time.Second,
		"Duration the LeaderElector clients should wait between tries of actions (duration string)")

	fs.StringVar(&watchNamespace, "namespace", "",
		"Namespace that the controller watches to reconcile cluster-api objects. If unspecified, the controller watches for cluster-api objects across all namespaces.")

	fs.StringVar(&profilerAddress, "profiler-address", "",
		"Bind address to expose the pprof profiler (e.g. localhost:6060)")

	fs.IntVar(&machineConcurrency, "machine-concurrency", 10,
		"Number of machines to process simultaneously")

	fs.DurationVar(&syncPeriod, "sync-period", 10*time.Minute,
		"The minimum interval at which watched resources are reconciled (e.g. 15m)")

	fs.IntVar(&webhookPort, "webhook-port", 0,
		"Webhook Server port, disabled by default. When enabled, the manager will only work as webhook server, no reconcilers are installed.")

	fs.StringVar(&healthAddr, "health-addr", ":9440",
		"The address the health endpoint binds to.")
}

// nolint:gocognit
func main() {
	rand.Seed(time.Now().UnixNano())

	InitFlags(pflag.CommandLine)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	ctrl.SetLogger(klogr.New())

	if profilerAddress != "" {
		klog.Infof("Profiler listening for requests at %s", profilerAddress)
		go func() {
			klog.Info(http.ListenAndServe(profilerAddress, nil))
		}()
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "controller-leader-election-capi",
		LeaseDuration:      &leaderElectionLeaseDuration,
		RenewDeadline:      &leaderElectionRenewDeadline,
		RetryPeriod:        &leaderElectionRetryPeriod,
		Namespace:          watchNamespace,
		SyncPeriod:         &syncPeriod,
		NewClient: util.DelegatingClientFuncWithUncached(
			&corev1.ConfigMap{},
			&corev1.ConfigMapList{},
			&corev1.Secret{},
			&corev1.SecretList{},
		),
		Port:                   webhookPort,
		HealthProbeBindAddress: healthAddr,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Setup the context that's going to be used in controllers and for the manager.
	setupChecks(mgr)
	setupReconcilers(mgr)
	setupWebhooks(mgr)

	// +kubebuilder:scaffold:builder
	setupLog.Info("starting manager", "version", version.Get().String())
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func setupChecks(mgr ctrl.Manager) {
	if err := mgr.AddReadyzCheck("ping", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to create ready check")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("ping", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to create health check")
		os.Exit(1)
	}
}

func setupReconcilers(mgr ctrl.Manager) {
	if webhookPort != 0 {
		return
	}

	if err := (&controllers.MachineReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("BYOMachine"),
	}).SetupWithManager(mgr, concurrency(machineConcurrency)); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "BYOMachine")
		os.Exit(1)
	}
}

func setupWebhooks(mgr ctrl.Manager) {
	if webhookPort == 0 {
		return
	}

	if err := (&infrav1.BYOMachine{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "BYOMachine")
		os.Exit(1)
	}

	if err := (&infrav1.BYOMachineTemplate{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "BYOMachineTemplate")
		os.Exit(1)
	}
}

func concurrency(c int) controller.Options {
	return controller.Options{MaxConcurrentReconciles: c}
}
