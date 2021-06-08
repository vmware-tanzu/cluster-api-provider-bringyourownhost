package main

import (
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/api/v1alpha4"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	kcapi "k8s.io/client-go/tools/clientcmd/api"
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
)

func TestHostAgent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HostAgent Suite")
}

var _ = BeforeSuite(func() {
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "config", "crd", "bases"),
			filepath.Join(build.Default.GOPATH, "pkg", "mod", "sigs.k8s.io", "cluster-api@v0.3.11-0.20210524195020-fca52981fe0a", "config", "crd", "bases"),
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
	writeKubeConfig(cfg)

	pathToHostAgentBinary, err = gexec.Build("github.com/vmware-tanzu/cluster-api-provider-byoh/host_agent")
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
	os.Remove(kubeconfigFile.Name())
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

func writeKubeConfig(cfg *rest.Config) {

	kubeconfigFile, err = ioutil.TempFile("", tmpFilePrefix)
	Expect(err).NotTo(HaveOccurred())

	kubeConfig := kcapi.NewConfig()
	kubeConfig.Clusters[clusterName] = &kcapi.Cluster{
		Server: fmt.Sprintf("http://%s", cfg.Host),
	}
	kcCtx := kcapi.NewContext()
	kcCtx.Cluster = clusterName
	kubeConfig.Contexts[clusterName] = kcCtx
	kubeConfig.CurrentContext = clusterName

	defer kubeconfigFile.Close()

	contents, err := clientcmd.Write(*kubeConfig)
	Expect(err).NotTo(HaveOccurred())

	amt, err := kubeconfigFile.Write(contents)
	Expect(err).NotTo(HaveOccurred())
	Expect(contents).To(HaveLen(amt))
}
