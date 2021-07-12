package main

import (
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/api/v1alpha4"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var (
	err                   error
	pathToHostAgentBinary string
	kubeconfigFile        *os.File
	cfg                   *rest.Config
	k8sClient             client.Client
	tmpFilePrefix         = "kubeconfigFile-"
	clusterName           = "test-cluster"
	testEnv               *envtest.Environment
	defaultNamespace      string = "default"
)

func TestHostAgent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Agent Suite")
}

var _ = BeforeSuite(func() {
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "config", "crd", "bases"),
			filepath.Join(build.Default.GOPATH, "pkg", "mod", "sigs.k8s.io", "cluster-api@v0.4.0", "config", "crd", "bases"),
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
	writeKubeConfig()

	pathToHostAgentBinary, err = gexec.Build("github.com/vmware-tanzu/cluster-api-provider-byoh/agent")
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
	os.Remove(kubeconfigFile.Name())
	gexec.TerminateAndWait(time.Duration(10) * time.Second)
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

func writeKubeConfig() {
	kubeconfigFile, err = ioutil.TempFile("", tmpFilePrefix)
	Expect(err).NotTo(HaveOccurred())

	user, err1 := testEnv.ControlPlane.AddUser(envtest.User{
		Name:   "envtest-admin",
		Groups: []string{"system:masters"},
	}, nil)
	Expect(err1).NotTo(HaveOccurred())

	kubeConfig, err2 := user.KubeConfig()
	Expect(err2).NotTo(HaveOccurred())

	_, err = kubeconfigFile.Write(kubeConfig)
	Expect(err).NotTo(HaveOccurred())
	defer kubeconfigFile.Close()
}
