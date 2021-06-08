package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"os/exec"
	"strings"

	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/api/v1alpha4"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	hostName  string = "jaime.com"
	namespace string
	scheme    *runtime.Scheme
)

func init() {
	scheme = runtime.NewScheme()
	infrastructurev1alpha4.AddToScheme(scheme)
	corev1.AddToScheme(scheme)
	clusterv1.AddToScheme(scheme)

	flag.StringVar(&namespace, "namespace", "default", "Namespace in the management cluster where you would like to register this host")
}

type HostReconciler struct {
	Client client.Client
}

func (r HostReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	machine := &clusterv1.Machine{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: "test-machine", Namespace: namespace}, machine)
	if err != nil {
		klog.Fatal(err)
	}

	secret := &corev1.Secret{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: *machine.Spec.Bootstrap.DataSecretName, Namespace: namespace}, secret)
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
	err = r.Client.Get(ctx, types.NamespacedName{Name: hostName, Namespace: namespace}, byoHost)
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

func main() {
	flag.Parse()

	config, err := ctrl.GetConfig()
	if err != nil {
		klog.Fatal(err)
	}

	k8sClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		klog.Fatal(err)
	}

	byoHost := &infrastructurev1alpha4.ByoHost{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ByoHost",
			APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      hostName,
			Namespace: namespace,
		},
		Spec: infrastructurev1alpha4.ByoHostSpec{},
	}

	err = k8sClient.Create(context.TODO(), byoHost)

	if err != nil {
		klog.Fatal(err)
	}

	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme: scheme,
	})
	if err != nil {
		klog.Fatal(err, "unable to start manager")
	}

	reconciler := HostReconciler{Client: k8sClient}
	ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1alpha4.ByoHost{}).
		Complete(reconciler)

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		klog.Fatal(err, "problem running manager")
	}
}
