// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var byoclusterlog = logf.Log.WithName("byocluster-resource")

func (r *ByoCluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-infrastructure-cluster-x-k8s-io-v1beta1-byocluster,mutating=true,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=byoclusters,verbs=create;update,versions=v1beta1,name=mbyocluster.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &ByoCluster{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
// nolint: stylecheck
func (r *ByoCluster) Default() {
	byoclusterlog.Info("default", "name", r.Name)

	if r.Spec.BundleLookupTag == "" {
		r.Spec.BundleLookupTag = "v0.1.0_alpha.2"
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-infrastructure-cluster-x-k8s-io-v1beta1-byocluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=byoclusters,verbs=create;update,versions=v1beta1,name=vbyocluster.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &ByoCluster{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ByoCluster) ValidateCreate() error {
	byoclusterlog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ByoCluster) ValidateUpdate(old runtime.Object) error {
	byoclusterlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ByoCluster) ValidateDelete() error {
	byoclusterlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
