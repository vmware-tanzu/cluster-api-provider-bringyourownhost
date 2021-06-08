package reconciler

import (
	"context"
	"encoding/base64"
	"fmt"
	"os/exec"
	"strings"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/api/v1alpha4"
	corev1 "k8s.io/api/core/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	ctrl "sigs.k8s.io/controller-runtime"
)

type HostReconciler struct {
	Client client.Client
}

func (r HostReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	machine := &clusterv1.Machine{}

	err := r.Client.Get(ctx, types.NamespacedName{Name: "test-machine", Namespace: req.Namespace}, machine)
	if err != nil {
		klog.Fatal(err)
	}

	secret := &corev1.Secret{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: *machine.Spec.Bootstrap.DataSecretName, Namespace: req.Namespace}, secret)
	if err != nil {
		klog.Fatal(err)
	}
	bootstrapScript := secret.Data["value"]

	decodedScript, err := base64.StdEncoding.DecodeString(string(bootstrapScript))
	if err != nil {
		klog.Fatal(err)
	}
	commands := strings.Split(string(decodedScript), " ")

	cmd := exec.Command(commands[0], strings.Join(commands[1:], " "))
	out, err := cmd.Output()

	if err != nil {
		klog.Fatal(err)
	}
	fmt.Println(string(out))

	byoHost := &infrastructurev1alpha4.ByoHost{}
	err = r.Client.Get(ctx, req.NamespacedName, byoHost)
	if err != nil {
		klog.Fatal(err)
	}

	helper, err := patch.NewHelper(byoHost, r.Client)
	if err != nil {
		klog.Fatal(err)
	}
	conditions.MarkTrue(byoHost, infrastructurev1alpha4.K8sComponentsInstalledCondition)
	err = helper.Patch(ctx, byoHost)
	if err != nil {
		klog.Fatal(err)
	}

	return ctrl.Result{}, nil
}
