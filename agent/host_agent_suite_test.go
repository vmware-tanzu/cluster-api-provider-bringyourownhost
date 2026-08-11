// Copyright 2021 VMware, Inc. All Rights Reserved.
// Copyright 2026 Platform9, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// nolint: nolintlint,testpackage
package main

import (
	"context"
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	goruntime "runtime"
	"testing"
	"time"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	dClient "github.com/docker/docker/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	infrastructurev1beta1 "github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/apis/infrastructure/v1beta1"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/test/e2e"
	certv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientset "k8s.io/client-go/kubernetes"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	// pathToHostAgentBinary is built for the native OS/arch: help_flag_test.go
	// executes it directly as a local subprocess.
	pathToHostAgentBinary string
	// pathToLinuxHostAgentBinary is always linux/<host arch>: it gets copied
	// into Linux BYOH host containers and executed there (see setupTestInfra),
	// regardless of the OS the test suite itself runs on (e.g. macOS).
	pathToLinuxHostAgentBinary string
	kubeconfigFile             *os.File
	k8sClient                  client.Client
	clientSet                  clientset.Interface
	tmpFilePrefix              = "kubeconfigFile-"
	defaultByoMachineName      = "my-byomachine"
	agentLogFile               = "/tmp/agent-integration.log"
	execLogFile                = "/tmp/agent-exec.log"
	fakeDownloadPath           = "fake-download-path"
	fakeBootstrapSecret        = "fake-bootstrap-secret"
	fakeInstallationSecret     = "fake-installation-secret"
	testEnv                    *envtest.Environment
	dockerClient               *dClient.Client
	// hostNameSuffix is a per-run Unix timestamp appended to os.Hostname() to make
	// container names unique across test runs, preventing conflicts from stale containers.
	hostNameSuffix string
)

const (
	bundleLookupBaseRegistry = "projects.registry.vmware.com/cluster_api_provider_bringyourownhost"
	K8sVersion               = "v1.22.3"
)

func TestHostAgent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Agent Suite")
}

func getKubeConfig() *os.File {
	return kubeconfigFile
}

func setKubeConfig(kubeConfig *os.File) {
	kubeconfigFile = kubeConfig
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	hostNameSuffix = fmt.Sprintf("%d", time.Now().Unix())
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "config", "crd", "bases"),
			filepath.Join(build.Default.GOPATH, "pkg", "mod", "sigs.k8s.io", "cluster-api@v1.4.4", "config", "crd", "bases"),
			filepath.Join(build.Default.GOPATH, "pkg", "mod", "sigs.k8s.io", "cluster-api@v1.4.4", "bootstrap", "kubeadm", "config", "crd", "bases"),
		},

		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	scheme := runtime.NewScheme()

	err = infrastructurev1beta1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = corev1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = clusterv1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = certv1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())

	clientSet = clientset.NewForConfigOrDie(cfg)

	dockerClient, err = dClient.NewClientWithOpts(dClient.FromEnv, dClient.WithAPIVersionNegotiation())
	Expect(err).NotTo(HaveOccurred())

	// The production build always uses CGO_ENABLED=0 because the agent must run
	// on arbitrary Linux hosts without dynamic library dependencies. Match that here.
	pathToHostAgentBinary, err = gexec.BuildWithEnvironment(
		"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent",
		[]string{"CGO_ENABLED=0"},
	)
	Expect(err).NotTo(HaveOccurred())

	pathToLinuxHostAgentBinary, err = gexec.BuildWithEnvironment(
		"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent",
		[]string{"CGO_ENABLED=0", "GOOS=linux", "GOARCH=" + goruntime.GOARCH},
	)
	Expect(err).NotTo(HaveOccurred())

	writeKubeConfig()
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
	err := os.Remove(getKubeConfig().Name())
	Expect(err).NotTo(HaveOccurred())
	gexec.TerminateAndWait(time.Duration(10) * time.Second)
	err = testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func writeKubeConfig() {
	kubeConf, err := os.CreateTemp("", tmpFilePrefix)
	Expect(err).NotTo(HaveOccurred())
	setKubeConfig(kubeConf)

	defer func(config *os.File) {
		_ = config.Close()
	}(getKubeConfig())

	user, err := testEnv.ControlPlane.AddUser(envtest.User{
		Name:   "envtest-admin",
		Groups: []string{"system:masters"},
	}, nil)
	Expect(err).NotTo(HaveOccurred())

	kubeConfigData, err := user.KubeConfig()
	Expect(err).NotTo(HaveOccurred())

	_, err = getKubeConfig().Write(kubeConfigData)
	Expect(err).NotTo(HaveOccurred())
}

func setupTestInfra(ctx context.Context, hostname, kubeconfig string, namespace *corev1.Namespace) *e2e.ByoHostRunner {
	byohostRunner := e2e.ByoHostRunner{
		Context:               ctx,
		Namespace:             namespace.Name,
		PathToHostAgentBinary: pathToLinuxHostAgentBinary,
		DockerClient:          dockerClient,
		NetworkInterface:      "host",
		ByoHostName:           hostname,
		Port:                  testEnv.ControlPlane.APIServer.Port,
		CommandArgs: map[string]string{
			"--namespace": namespace.Name,
			"-v":          "1",
		},
		KubeconfigFile: kubeconfig,
	}

	return &byohostRunner
}

func cleanup(ctx context.Context, byoHostContainer *container.CreateResponse, namespace *corev1.Namespace, agentLogFile string) {
	// byoHostContainer is nil when BeforeEach failed before the container was created.
	if byoHostContainer != nil {
		err := dockerClient.ContainerStop(ctx, byoHostContainer.ID, container.StopOptions{})
		Expect(err).NotTo(HaveOccurred())

		// Force ensures removal succeeds even when a lingering exec attachment hasn't
		// fully closed on the daemon side after ContainerStop.
		err = dockerClient.ContainerRemove(ctx, byoHostContainer.ID, dockertypes.ContainerRemoveOptions{Force: true})
		Expect(err).NotTo(HaveOccurred())
	}

	err := k8sClient.Delete(ctx, namespace)
	Expect(err).NotTo(HaveOccurred(), "failed to delete test namespace")

	_, err = os.Stat(agentLogFile)
	if err == nil {
		err = os.Remove(agentLogFile)
		if err != nil {
			e2e.Showf("error removing log file %s: %v", agentLogFile, err)
		}
	}
	_, err = os.Stat(e2e.TempKubeconfigPath)
	if err == nil {
		err = os.Remove(e2e.TempKubeconfigPath)
		if err != nil {
			e2e.Showf("error removing kubeconfig file %s: %v", e2e.TempKubeconfigPath, err)
		}
	}
}
