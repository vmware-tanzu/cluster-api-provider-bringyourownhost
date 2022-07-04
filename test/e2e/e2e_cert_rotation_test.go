// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// nolint: testpackage
package e2e

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
)

// creating a workload cluster
// This test is meant to provide a first, fast signal to detect regression; it is recommended to use it as a PR blocker test.
var _ = FDescribe("When BYOH joins existing cluster [PR-Blocking]", func() {

	var (
		ctx                 context.Context
		specName            = "quick-start"
		namespace           *corev1.Namespace
		cancelWatches       context.CancelFunc
		clusterResources    *clusterctl.ApplyClusterTemplateAndWaitResult
		dockerClient        *client.Client
		err                 error
		byohostContainerIDs []string
		agentLogFile        = "/tmp/host-agent.log"
		byoHostName         = "byohost"
	)

	BeforeEach(func() {

		ctx = context.TODO()
		Expect(ctx).NotTo(BeNil(), "ctx is required for %s spec", specName)

		Expect(e2eConfig).NotTo(BeNil(), "Invalid argument. e2eConfig can't be nil when calling %s spec", specName)
		Expect(clusterctlConfigPath).To(BeAnExistingFile(), "Invalid argument. clusterctlConfigPath must be an existing file when calling %s spec", specName)
		Expect(bootstrapClusterProxy).NotTo(BeNil(), "Invalid argument. bootstrapClusterProxy can't be nil when calling %s spec", specName)
		Expect(os.MkdirAll(artifactFolder, 0755)).To(Succeed(), "Invalid argument. artifactFolder can't be created for %s spec", specName)

		Expect(e2eConfig.Variables).To(HaveKey(KubernetesVersion))

		// set up a Namespace where to host objects for this spec and create a watcher for the namespace events.
		namespace, cancelWatches = setupSpecNamespace(ctx, specName, bootstrapClusterProxy, artifactFolder)
		clusterResources = new(clusterctl.ApplyClusterTemplateAndWaitResult)
	})

	It("Should create a workload cluster with single BYOH host", func() {
		dockerClient, err = client.NewClientWithOpts(client.FromEnv)
		Expect(err).NotTo(HaveOccurred())

		runner := ByoHostRunner{
			Context:               ctx,
			clusterConName:        clusterConName,
			Namespace:             namespace.Name,
			PathToHostAgentBinary: pathToHostAgentBinary,
			DockerClient:          dockerClient,
			NetworkInterface:      "kind",
			bootstrapClusterProxy: bootstrapClusterProxy,
			CommandArgs: map[string]string{
				"--bootstrap-kubeconfig": "/bootstrap.conf",
				"--namespace":            namespace.Name,
				"--v":                    "1",
				"--certExpiryDuration":   "601",
			},
		}

		var output types.HijackedResponse
		runner.ByoHostName = byoHostName
		runner.BootstrapKubeconfigData = generateBootstrapKubeconfig(runner.Context, bootstrapClusterProxy, clusterConName)
		byohost, err := runner.SetupByoDockerHost()
		Expect(err).NotTo(HaveOccurred())

		output, byohostContainerID, err := runner.ExecByoDockerHost(byohost)
		Expect(err).NotTo(HaveOccurred())

		defer output.Close()
		byohostContainerIDs = append(byohostContainerIDs, byohostContainerID)
		f := WriteDockerLog(output, agentLogFile)
		defer func() {
			deferredErr := f.Close()
			if deferredErr != nil {
				Showf("error closing file %s: %v", agentLogFile, deferredErr)
			}
		}()

		Eventually(func() (done bool) {
			_, err := os.Stat(agentLogFile)
			if err == nil {
				data, err := os.ReadFile(agentLogFile)
				fmt.Println(string(data))
				if err == nil && strings.Contains(string(data), "\"msg\"=\"certificate expired. Creating new certificate.\"") {
					return true
				}
			}
			return false
		}, time.Minute*25).Should(BeTrue())
	})

	JustAfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			ShowInfo([]string{agentLogFile})
		}
	})

	AfterEach(func() {
		// Dumps all the resources in the spec namespace, then cleanups the cluster object and the spec namespace itself.
		dumpSpecResourcesAndCleanup(ctx, specName, bootstrapClusterProxy, artifactFolder, namespace, cancelWatches, clusterResources.Cluster, e2eConfig.GetIntervals, skipCleanup)

		if dockerClient != nil && len(byohostContainerIDs) != 0 {
			for _, byohostContainerID := range byohostContainerIDs {
				err := dockerClient.ContainerStop(ctx, byohostContainerID, nil)
				Expect(err).NotTo(HaveOccurred())

				err = dockerClient.ContainerRemove(ctx, byohostContainerID, types.ContainerRemoveOptions{})
				Expect(err).NotTo(HaveOccurred())
			}
		}

		err := os.Remove(agentLogFile)
		if err != nil {
			Showf("error removing file %s: %v", agentLogFile, err)
		}

		err = os.Remove(ReadByohControllerManagerLogShellFile)
		if err != nil {
			Showf("error removing file %s: %v", ReadByohControllerManagerLogShellFile, err)
		}

		err = os.Remove(ReadAllPodsShellFile)
		if err != nil {
			Showf("error removing file %s: %v", ReadAllPodsShellFile, err)
		}
	})
})
