// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// nolint: testpackage
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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	infrastructurev1beta1 "sigs.k8s.io/cluster-api-provider-bringyourownhost/apis/infrastructure/v1beta1"
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
	testEnv               *envtest.Environment
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
			filepath.Join(build.Default.GOPATH, "pkg", "mod", "sigs.k8s.io", "cluster-api@v1.0.4", "config", "crd", "bases"),
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
	writeKubeConfig()

	pathToHostAgentBinary, err = gexec.Build("sigs.k8s.io/cluster-api-provider-bringyourownhost/agent")
	Expect(err).NotTo(HaveOccurred())
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
