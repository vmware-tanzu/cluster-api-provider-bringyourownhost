// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package authenticator_test

import (
	"context"
	"crypto/rsa"
	"go/build"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/authenticator"
	certv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func TestAuthenticator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Authenticator Suite")
}

var (
	ns                     = "test-ns"
	hostName               = "test-host"
	cfg                    *rest.Config
	k8sClient              client.Client
	k8sManager             manager.Manager
	bootstrapAuthenticator *authenticator.BootstrapAuthenticator
	testEnv                *envtest.Environment
)

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
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

	err = corev1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = certv1.AddToScheme(scheme)
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

	bootstrapAuthenticator = &authenticator.BootstrapAuthenticator{
		Client:     k8sManager.GetClient(),
		HostName:   hostName,
		PrivateKey: &rsa.PrivateKey{},
	}
	err = bootstrapAuthenticator.SetupWithManager(context.TODO(), k8sManager)
	Expect(err).NotTo(HaveOccurred())
	go func() {
		err = k8sManager.GetCache().Start(context.TODO())
		Expect(err).NotTo(HaveOccurred())
	}()

	Expect(k8sManager.GetCache().WaitForCacheSync(context.TODO())).To(BeTrue())
})

var _ = AfterSuite(func() {
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

func WaitForObjectsToBePopulatedInCache(objects ...client.Object) {
	for _, object := range objects {
		objectCopy := object.DeepCopyObject().(client.Object)
		key := client.ObjectKeyFromObject(object)
		Eventually(func() (done bool) {
			if err := bootstrapAuthenticator.Client.Get(context.TODO(), key, objectCopy); err != nil {
				return false
			}
			return true
		}).Should(BeTrue())
	}
}
