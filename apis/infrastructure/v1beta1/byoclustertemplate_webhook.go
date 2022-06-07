// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func (byoClusterTemplate *ByoClusterTemplate) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(byoClusterTemplate).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-infrastructure-cluster-x-k8s-io-v1beta1-byoclustertemplate,mutating=true,failurePolicy=fail,matchPolicy=Equivalent,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=byoclustertemplates,verbs=create;update,versions=v1beta1,name=mbyoclustertemplate.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &ByoClusterTemplate{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (byoClusterTemplate *ByoClusterTemplate) Default() {
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-infrastructure-cluster-x-k8s-io-v1beta1-byoclustertemplate,mutating=false,failurePolicy=fail,matchPolicy=Equivalent,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=byoclustertemplates,verbs=create;update,versions=v1beta1,name=vbyoclustertemplate.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &ByoClusterTemplate{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (byoClusterTemplate *ByoClusterTemplate) ValidateCreate() error {
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (byoClusterTemplate *ByoClusterTemplate) ValidateUpdate(oldRaw runtime.Object) error {
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (byoClusterTemplate *ByoClusterTemplate) ValidateDelete() error {
	return nil
}
