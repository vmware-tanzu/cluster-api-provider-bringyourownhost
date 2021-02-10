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
	"context"
	"flag"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	infrav1 "vmware-tanzu/cluster-api-provider-byoh/api/v1alpha3"
	"vmware-tanzu/cluster-api-provider-byoh/version"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog"
	"k8s.io/klog/klogr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")

	// flags
	metricsAddr     string
	profilerAddress string
	syncPeriod      time.Duration
	healthAddr      string
	labels          []string
)

func init() {
	klog.InitFlags(nil)

	_ = clientgoscheme.AddToScheme(scheme)
	_ = clusterv1.AddToScheme(scheme)
	_ = infrav1.AddToScheme(scheme)
}

// InitFlags initializes the flags.
func InitFlags(fs *pflag.FlagSet) {

	fs.StringArrayVar(&labels, "labels", nil,
		"Labels identifiyng the node.")

	// TODO might be we want a flag to declare if it is managed or not; for the time being, we are forcing unmanaged

	fs.StringVar(&metricsAddr, "metrics-addr", ":8080",
		"The address the metric endpoint binds to.")

	fs.StringVar(&profilerAddress, "profiler-address", "",
		"Bind address to expose the pprof profiler (e.g. localhost:6060)")

	fs.DurationVar(&syncPeriod, "sync-period", 10*time.Minute,
		"The minimum interval at which watched resources are reconciled (e.g. 15m)")

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

	// TODO: validate kubeconfig

	if profilerAddress != "" {
		klog.Infof("Profiler listening for requests at %s", profilerAddress)
		go func() {
			klog.Info(http.ListenAndServe(profilerAddress, nil))
		}()
	}

	setupLog.Info("manager", "version", version.Get().String())
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		SyncPeriod:         &syncPeriod,
		NewClient: util.DelegatingClientFuncWithUncached(
			&infrav1.BYOMachine{},
		),
		HealthProbeBindAddress: healthAddr,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Register the BYOHost into the available capacity before starting the manager/reconcile loop.
	host, err := HostRegistration(mgr)
	if err != nil {
		setupLog.Error(err, "failed to register the host")
		os.Exit(1)
	}

	if err := (&HostReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("BYOHost"),
		Host:   host,
	}).SetupWithManager(mgr, controller.Options{}); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "BYOHost")
		os.Exit(1)
	}

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func HostRegistration(mgr ctrl.Manager) (*infrav1.BYOHost, error) {
	// TODO implement label parse & validation
	// TODO implement auto discovery of managed/kubernetes version
	l := map[string]string{}

	// Note: we are using the same logic used by kubeadm for assigning node name.
	nodeName, err := os.Hostname()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't determine hostname")
	}
	host := &infrav1.BYOHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:   nodeName,
			Labels: l,
		},
	}

	if err := mgr.GetClient().Create(context.TODO(), host); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return nil, errors.Wrapf(err, "failed to create BYOHost")
		}

		// Reads the existing host using a new client without cache, given that the cache does not exists yet.
		c, err := client.New(mgr.GetConfig(), client.Options{Scheme: mgr.GetScheme()})
		if err != nil {
			return nil, err
		}

		if err := c.Get(context.TODO(), client.ObjectKey{Name: nodeName}, host); err != nil {
			return nil, errors.Wrapf(err, "failed to the host BYOHost")
		}
		// TODO: should we update in case the host already exists? e.g. change of labels
	}
	setupLog.Info("BYOHost registered", "Name", host.Name)
	// TODO log labels.

	return host, nil
}
