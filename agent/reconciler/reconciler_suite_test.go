package reconciler

import (
	"context"
	"go/build"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/apis/infrastructure/v1alpha4"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
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
	err         error
	cfg         *rest.Config
	k8sClient   client.Client
	k8sManager  manager.Manager
	patchHelper *patch.Helper
	reconciler  *HostReconciler
	testEnv     *envtest.Environment
)

var _ = BeforeSuite(func() {
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "config", "crd", "bases"),
			filepath.Join(build.Default.GOPATH, "pkg", "mod", "sigs.k8s.io", "cluster-api@v0.4.2", "config", "crd", "bases"),
		},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	scheme := runtime.NewScheme()

	err = infrastructurev1alpha4.AddToScheme(scheme)
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

	reconciler = &HostReconciler{
		Client: k8sClient,
	}
	err = reconciler.SetupWithManager(context.TODO(), k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// TODO: Either delete the below goroutine
	// or move to individual test contexts where controller-runtime is required

	// go func() {
	// 	err = k8sManager.Start(ctrl.SetupSignalHandler())
	// 	Expect(err).NotTo(HaveOccurred())
	// }()
})

var _ = AfterSuite(func() {
	err = testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})
