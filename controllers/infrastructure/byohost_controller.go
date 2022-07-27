// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"strconv"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/prometheus/client_golang/prometheus"
	infrastructurev1beta1 "github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/apis/infrastructure/v1beta1"
)

// ByoHostReconciler reconciles a ByoHost object
type ByoHostReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var (
	memoryUsageMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "memory_usage",
	})

	memoryTotalMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "memory_total",
	})

	cpuUsageMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "cpu_usage",
	})

	cpuCoreMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "cpu_cores",
	})
)

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=byohosts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=byohosts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=byohosts/finalizers,verbs=update
//+kubebuilder:rbac:groups=certificates.k8s.io,resources=certificatesigningrequests,verbs=create;get;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ByoHost object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *ByoHostReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	_ = log.FromContext(ctx)
	// TODO-OBSERVABILITY - Task3
	// Add logic behind an environment flag to deduce/update health of a machine based
	// on last health status reported time

	// TODO-OBSERVABILITY - Task4
	// Expose static, runtime resource footprint metrics
	byoHost := &infrastructurev1beta1.ByoHost{}
	err := r.Client.Get(ctx, req.NamespacedName, byoHost)
	if err != nil {
		return ctrl.Result{}, err
	}
	totalMemory, _ := strconv.ParseFloat(byoHost.Status.HostDetails.Memory1, 64)
	memoryTotalMetric.Set(totalMemory)

	usedMemory, _ := strconv.ParseFloat(byoHost.Status.HostDetails.Memory2, 64)
	memoryUsageMetric.Set(usedMemory)

	cpuCores, _ := strconv.ParseFloat(byoHost.Status.HostDetails.CPU1, 64)
	cpuCoreMetric.Set(cpuCores)

	cpuUsage, _ := strconv.ParseFloat(byoHost.Status.HostDetails.CPU2, 64)
	cpuUsageMetric.Set(cpuUsage)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ByoHostReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1beta1.ByoHost{}).
		Complete(r)
}

func init() {
	metrics.Registry.MustRegister(memoryUsageMetric)
	metrics.Registry.MustRegister(memoryTotalMetric)
	metrics.Registry.MustRegister(cpuCoreMetric)
	metrics.Registry.MustRegister(cpuUsageMetric)
}
