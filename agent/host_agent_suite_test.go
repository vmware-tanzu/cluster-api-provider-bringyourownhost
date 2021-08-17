// Copyright 2021 VMware, Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

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
	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/apis/infrastructure/v1alpha4"
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
	defaultClusterName    = "default-test-cluster"
	testEnv               *envtest.Environment
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
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

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
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
	os.Remove(kubeconfigFile.Name())
	gexec.TerminateAndWait(time.Duration(10) * time.Second)
	err = testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func writeKubeConfig() {
	kubeconfigFile, err = ioutil.TempFile("", tmpFilePrefix)
	Expect(err).NotTo(HaveOccurred())

	user, err := testEnv.ControlPlane.AddUser(envtest.User{
		Name:   "envtest-admin",
		Groups: []string{"system:masters"},
	}, nil)
	Expect(err).NotTo(HaveOccurred())

	kubeConfig, err := user.KubeConfig()
	Expect(err).NotTo(HaveOccurred())

	_, err = kubeconfigFile.Write(kubeConfig)
	Expect(err).NotTo(HaveOccurred())
	defer kubeconfigFile.Close()
}
