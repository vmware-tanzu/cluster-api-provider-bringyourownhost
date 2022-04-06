// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// nolint: testpackage
package main

import (
	"context"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	dClient "github.com/docker/docker/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	infrastructurev1beta1 "github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/apis/infrastructure/v1beta1"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/test/e2e"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	pathToHostAgentBinary string
	kubeconfigFile        *os.File
	k8sClient             client.Client
	tmpFilePrefix         = "kubeconfigFile-"
	defaultByoMachineName = "my-byomachine"
	agentLogFile          = "/tmp/agent-integration.log"
	fakeKubeConfig        = "fake-kubeconfig-path"
	fakeDownloadPath      = "fake-download-path"
	fakeBootstrapSecret   = "fake-bootstrap-secret"
	testEnv               *envtest.Environment
	dockerClient          *dClient.Client
)

const (
	bundleLookupBaseRegistry = "projects.registry.vmware.com/cluster_api_provider_bringyourownhost"
	BundleLookupTag          = "v1.22.3"
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
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "config", "crd", "bases"),
			filepath.Join(build.Default.GOPATH, "pkg", "mod", "sigs.k8s.io", "cluster-api@v1.1.3", "config", "crd", "bases"),
			filepath.Join(build.Default.GOPATH, "pkg", "mod", "sigs.k8s.io", "cluster-api@v1.1.3", "bootstrap", "kubeadm", "config", "crd", "bases"),
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

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())

	dockerClient, err = dClient.NewClientWithOpts(dClient.FromEnv)
	Expect(err).NotTo(HaveOccurred())

	pathToHostAgentBinary, err = gexec.Build("github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent")
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
	kubeConf, err := ioutil.TempFile("", tmpFilePrefix)
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
		PathToHostAgentBinary: pathToHostAgentBinary,
		DockerClient:          dockerClient,
		NetworkInterface:      "host",
		ByoHostName:           hostname,
		Port:                  testEnv.ControlPlane.APIServer.Port,
		CommandArgs: map[string]string{
			"--kubeconfig": "/mgmt.conf",
			"--namespace":  namespace.Name,
			"-v":           "1",
		},
		KubeconfigFile: kubeconfig,
	}

	return &byohostRunner
}

func cleanup(ctx context.Context, byoHostContainer *container.ContainerCreateCreatedBody, namespace *corev1.Namespace, agentLogFile string) {
	err := dockerClient.ContainerStop(ctx, byoHostContainer.ID, nil)
	Expect(err).NotTo(HaveOccurred())

	err = dockerClient.ContainerRemove(ctx, byoHostContainer.ID, dockertypes.ContainerRemoveOptions{})
	Expect(err).NotTo(HaveOccurred())

	err = k8sClient.Delete(ctx, namespace)
	Expect(err).NotTo(HaveOccurred(), "failed to delete test namespace")

	_, err = os.Stat(agentLogFile)
	if err == nil {
		err = os.Remove(agentLogFile)
		if err != nil {
			e2e.Showf("error removing log file %s: %v", agentLogFile, err)
		}
	}
}
