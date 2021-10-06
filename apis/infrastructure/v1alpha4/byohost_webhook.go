/*
Copyright 2021.

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

package v1alpha4

import (
	"errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package
var byohostlog = logf.Log.WithName("byohost-resource")

func (h *ByoHost) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(h).
		Complete()
}

//+kubebuilder:webhook:path=/validate-infrastructure-cluster-x-k8s-io-v1alpha4-byohost,mutating=false,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=byohosts,verbs=create;update;delete,versions=v1alpha4,name=vbyohost.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &ByoHost{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (h *ByoHost) ValidateCreate() error {
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (h *ByoHost) ValidateUpdate(old runtime.Object) error {
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (h *ByoHost) ValidateDelete() error {
	byohostlog.Info("validate delete", "name", h.Name)
	groupResource := schema.GroupResource{Group: "infrastructure.cluster.x-k8s.io", Resource: "byohost"}

	if h.Status.MachineRef != nil {
		return apierrors.NewForbidden(groupResource, h.Name, errors.New("cannot delete ByoHost when MachineRef is assigned"))
	}

	return nil
}
