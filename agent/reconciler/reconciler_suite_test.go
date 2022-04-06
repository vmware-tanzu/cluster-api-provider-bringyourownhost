// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package reconciler_test

import (
	"go/build"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/cloudinit/cloudinitfakes"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/reconciler"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/reconciler/reconcilerfakes"
	infrastructurev1beta1 "github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/apis/infrastructure/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func TestReconciler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Reconciler Suite")
}

var (
	cfg                *rest.Config
	k8sClient          client.Client
	k8sManager         manager.Manager
	patchHelper        *patch.Helper
	hostReconciler     *reconciler.HostReconciler
	testEnv            *envtest.Environment
	fakeCommandRunner  *cloudinitfakes.FakeICmdRunner
	fakeFileWriter     *cloudinitfakes.FakeIFileWriter
	fakeTemplateParser *cloudinitfakes.FakeITemplateParser
	fakeInstaller      *reconcilerfakes.FakeIK8sInstaller
)

var _ = BeforeSuite(func() {
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "config", "crd", "bases"),
			filepath.Join(build.Default.GOPATH, "pkg", "mod", "sigs.k8s.io", "cluster-api@v1.1.3", "config", "crd", "bases"),
		},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	scheme := runtime.NewScheme()

	err = infrastructurev1beta1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = corev1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = clusterv1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	k8sManager, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: ":6090",
	})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sManager).ToNot(BeNil())
})

var _ = AfterSuite(func() {
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})
