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

package controllers

import (
	"context"
	"go/build"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/cluster-api/controllers/remote"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	"sigs.k8s.io/controller-runtime/pkg/log"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/api/v1alpha4"
	//+kubebuilder:scaffold:imports

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterapi "sigs.k8s.io/cluster-api/api/v1alpha4"
	bootstrapv1 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1alpha4"
	ctrl "sigs.k8s.io/controller-runtime"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg                   *rest.Config
	k8sClient             client.Client
	testEnv               *envtest.Environment
	clientFake            client.Client
	reconciler            *ByoMachineReconciler
	defaultClusterName    string = "my-cluster"
	defaultNodeName       string = "my-host"
	defaultByoHostName    string = "my-host"
	defaultByoMachineName string = "my-machine"
	defaultNamespace      string = "default"
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "config", "crd", "bases"),
			filepath.Join(build.Default.GOPATH, "pkg", "mod", "sigs.k8s.io", "cluster-api@v0.4.0", "config", "crd", "bases"),
			filepath.Join(build.Default.GOPATH, "pkg", "mod", "sigs.k8s.io", "cluster-api@v0.4.0", "bootstrap", "kubeadm", "config", "crd", "bases"),
		},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = infrastructurev1alpha4.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = clusterapi.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = bootstrapv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme.Scheme,
		MetricsBindAddress: ":6080",
	})
	Expect(err).ToNot(HaveOccurred())

	testCluster := &clusterapi.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "cluster.x-k8s.io/v1alpha4",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultClusterName,
			Namespace: defaultNamespace,
		},
		Spec: clusterapi.ClusterSpec{},
	}
	Expect(k8sClient.Create(context.Background(), testCluster)).Should(Succeed())

	node := &corev1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1alpha4",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultNodeName,
			Namespace: defaultNamespace,
		},
		Spec:   corev1.NodeSpec{},
		Status: corev1.NodeStatus{},
	}
	clientFake = fake.NewClientBuilder().WithObjects(
		testCluster,
		node,
	).Build()

	reconciler = &ByoMachineReconciler{
		Client:  k8sClient,
		Tracker: remote.NewTestClusterCacheTracker(log.NullLogger{}, clientFake, scheme.Scheme, client.ObjectKey{Name: testCluster.Name, Namespace: testCluster.Namespace}),
	}
	err = reconciler.SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())
	}()

}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func newByoMachine(byoMachineName string, byoMachineNamespace string, clusterName string) *infrastructurev1alpha4.ByoMachine {
	byoMachine := &infrastructurev1alpha4.ByoMachine{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ByoMachine",
			APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      byoMachineName,
			Namespace: byoMachineNamespace,
			Labels: map[string]string{
				clusterapi.ClusterLabelName: clusterName,
			},
		},
		Spec: infrastructurev1alpha4.ByoMachineSpec{},
	}
	return byoMachine
}

func newByoHost(byoHostName string, byoHostNamespace string) *infrastructurev1alpha4.ByoHost {
	byoHost := &infrastructurev1alpha4.ByoHost{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ByoHost",
			APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      byoHostName,
			Namespace: byoHostNamespace,
		},
		Spec: infrastructurev1alpha4.ByoHostSpec{},
	}
	return byoHost
}
