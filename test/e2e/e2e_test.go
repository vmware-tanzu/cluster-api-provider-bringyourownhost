// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util"
)

const (
	KubernetesVersion = "KUBERNETES_VERSION"
	CNIPath           = "CNI"
	CNIResources      = "CNI_RESOURCES"
	IPFamily          = "IP_FAMILY"
)

// creating a workload cluster
// This test is meant to provide a first, fast signal to detect regression; it is recommended to use it as a PR blocker test.
var _ = Describe("When BYOH joins existing cluster [PR-Blocking]", func() {

	var (
		ctx                 context.Context
		specName            = "quick-start"
		namespace           *corev1.Namespace
		clusterName         string
		cancelWatches       context.CancelFunc
		clusterResources    *clusterctl.ApplyClusterTemplateAndWaitResult
		dockerClient        *client.Client
		err                 error
		byohostContainerIDs []string
		agentLogFile1       = "/tmp/host-agent1.log"
		agentLogFile2       = "/tmp/host-agent2.log"
	)

	BeforeEach(func() {

		ctx = context.TODO()
		Expect(ctx).NotTo(BeNil(), "ctx is required for %s spec", specName)

		Expect(e2eConfig).NotTo(BeNil(), "Invalid argument. e2eConfig can't be nil when calling %s spec", specName)
		Expect(clusterctlConfigPath).To(BeAnExistingFile(), "Invalid argument. clusterctlConfigPath must be an existing file when calling %s spec", specName)
		Expect(bootstrapClusterProxy).NotTo(BeNil(), "Invalid argument. bootstrapClusterProxy can't be nil when calling %s spec", specName)
		Expect(os.MkdirAll(artifactFolder, 0755)).To(Succeed(), "Invalid argument. artifactFolder can't be created for %s spec", specName)

		Expect(e2eConfig.Variables).To(HaveKey(KubernetesVersion))

		// Setup a Namespace where to host objects for this spec and create a watcher for the namespace events.
		namespace, cancelWatches = setupSpecNamespace(ctx, specName, bootstrapClusterProxy, artifactFolder)
		clusterResources = new(clusterctl.ApplyClusterTemplateAndWaitResult)
	})

	It("Should create a workload cluster with single BYOH host", func() {
		clusterName = fmt.Sprintf("%s-%s", specName, util.RandomString(6))
		byoHostName1 := "byohost1"
		byoHostName2 := "byohost2"

		dockerClient, err = client.NewClientWithOpts(client.FromEnv)
		Expect(err).NotTo(HaveOccurred())

		var output types.HijackedResponse
		output, byohostContainerID, err := setupByoDockerHost(ctx, clusterConName, byoHostName1, namespace.Name, dockerClient, bootstrapClusterProxy)
		Expect(err).NotTo(HaveOccurred())
		defer output.Close()
		byohostContainerIDs = append(byohostContainerIDs, byohostContainerID)
		f := WriteDockerLog(output, agentLogFile1)
		defer f.Close()

		output, byohostContainerID, err = setupByoDockerHost(ctx, clusterConName, byoHostName2, namespace.Name, dockerClient, bootstrapClusterProxy)
		Expect(err).NotTo(HaveOccurred())
		defer output.Close()
		byohostContainerIDs = append(byohostContainerIDs, byohostContainerID)

		// read the log of host agent container in backend, and write it
		f = WriteDockerLog(output, agentLogFile2)
		defer f.Close()

		clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
			ClusterProxy: bootstrapClusterProxy,
			ConfigCluster: clusterctl.ConfigClusterInput{
				LogFolder:                filepath.Join(artifactFolder, "clusters", bootstrapClusterProxy.GetName()),
				ClusterctlConfigPath:     clusterctlConfigPath,
				KubeconfigPath:           bootstrapClusterProxy.GetKubeconfigPath(),
				InfrastructureProvider:   "byoh:v0.1.0",
				Flavor:                   clusterctl.DefaultFlavor,
				Namespace:                namespace.Name,
				ClusterName:              clusterName,
				KubernetesVersion:        e2eConfig.GetVariable(KubernetesVersion),
				ControlPlaneMachineCount: pointer.Int64Ptr(1),
				WorkerMachineCount:       pointer.Int64Ptr(1),
			},
			WaitForClusterIntervals:      e2eConfig.GetIntervals(specName, "wait-cluster"),
			WaitForControlPlaneIntervals: e2eConfig.GetIntervals(specName, "wait-control-plane"),
			WaitForMachineDeployments:    e2eConfig.GetIntervals(specName, "wait-worker-nodes"),
		}, clusterResources)

	})

	JustAfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			ShowInfo([]string{agentLogFile1, agentLogFile2})
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

		os.Remove(agentLogFile1)
		os.Remove(agentLogFile2)
		os.Remove(ReadByohControllerManagerLogShellFile)
		os.Remove(ReadAllPodsShellFile)
	})
})
