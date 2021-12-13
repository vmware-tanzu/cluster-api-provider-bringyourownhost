// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package
var byoclusterlog = logf.Log.WithName("byocluster-resource")

func (h *ByoCluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(h).
		Complete()
}

//+kubebuilder:webhook:path=/validate-infrastructure-cluster-x-k8s-io-v1beta1-byocluster,mutating=false,failurePolicy=fail,sideEffects=None,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=byoclusters,verbs=create;update,versions=v1beta1,name=vbyocluster.kb.io,admissionReviewVersions={v1,v1beta1}
//+kubebuilder:webhook:path=/mutate-infrastructure-cluster-x-k8s-io-v1beta1-byocluster,mutating=true,failurePolicy=fail,sideEffects=None,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=byoclusters,verbs=create;update,versions=v1beta1,name=mbyocluster.kb.io,admissionReviewVersions={v1,v1beta1}

var (
	_ webhook.Validator = &ByoCluster{}
	_ webhook.Defaulter = &ByoCluster{}
)

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (h *ByoCluster) ValidateCreate() error {
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (h *ByoCluster) ValidateUpdate(old runtime.Object) error {
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (h *ByoCluster) ValidateDelete() error {
	return nil
}

// Default will set default values for the type.
func (h *ByoCluster) Default() {
	byoclusterlog.Info("default", "name", h.Name)

	if h.Spec.BundleLookupTag == "" {
		h.Spec.BundleLookupTag = "v0.1.0_alpha.2"
	}
}
