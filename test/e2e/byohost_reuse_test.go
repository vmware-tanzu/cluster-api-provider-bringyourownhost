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
	infrastructurev1beta1 "github.com/vmware-tanzu/cluster-api-provider-byoh/apis/infrastructure/v1beta1"
	corev1 "k8s.io/api/core/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util"
)

var _ = Describe("When BYO Host rejoins the capacity pool", func() {

	var (
		ctx                 context.Context
		specName            = "byohost-reuse"
		namespace           *corev1.Namespace
		cancelWatches       context.CancelFunc
		clusterResources    *clusterctl.ApplyClusterTemplateAndWaitResult
		dockerClient        *client.Client
		err                 error
		byohostContainerIDs []string
		agentLogFile1       = "/tmp/host-agent1.log"
		agentLogFile2       = "/tmp/host-agent-reuse.log"
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

	It("Should reuse the same BYO Host after it is reset", func() {
		clusterName := fmt.Sprintf("%s-%s", specName, util.RandomString(6))
		byoHostName1 := "byohost-1"
		byoHostName2 := "byohost-for-reuse"

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

		By("Creating a cluster")
		clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
			ClusterProxy: bootstrapClusterProxy,
			ConfigCluster: clusterctl.ConfigClusterInput{
				LogFolder:                filepath.Join(artifactFolder, "clusters", bootstrapClusterProxy.GetName()),
				ClusterctlConfigPath:     clusterctlConfigPath,
				KubeconfigPath:           bootstrapClusterProxy.GetKubeconfigPath(),
				InfrastructureProvider:   clusterctl.DefaultInfrastructureProvider,
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

		// Assert on byohost cluster label to match clusterName
		byoHostLookupKey := k8stypes.NamespacedName{Name: byoHostName2, Namespace: namespace.Name}
		byoHostToBeReused := &infrastructurev1beta1.ByoHost{}
		Expect(bootstrapClusterProxy.GetClient().Get(ctx, byoHostLookupKey, byoHostToBeReused)).Should(Succeed())
		cluster, ok := byoHostToBeReused.Labels[clusterv1.ClusterLabelName]
		Expect(ok).To(BeTrue())
		Expect(cluster).To(Equal(clusterName))

		By("Delete the cluster and freeing the ByoHosts")
		framework.DeleteAllClustersAndWait(ctx, framework.DeleteAllClustersAndWaitInput{
			Client:    bootstrapClusterProxy.GetClient(),
			Namespace: namespace.Name,
		}, e2eConfig.GetIntervals(specName, "wait-delete-cluster")...)

		// Assert if cluster label is removed
		// This verifies that the byohost has rejoined the capacity pool
		byoHostToBeReused = &infrastructurev1beta1.ByoHost{}
		Expect(bootstrapClusterProxy.GetClient().Get(ctx, byoHostLookupKey, byoHostToBeReused)).Should(Succeed())
		_, ok = byoHostToBeReused.Labels[clusterv1.ClusterLabelName]
		Expect(ok).To(BeFalse())

		By("Creating a new cluster")
		clusterName = fmt.Sprintf("%s-%s", specName, util.RandomString(6))
		clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
			ClusterProxy: bootstrapClusterProxy,
			ConfigCluster: clusterctl.ConfigClusterInput{
				LogFolder:                filepath.Join(artifactFolder, "clusters", bootstrapClusterProxy.GetName()),
				ClusterctlConfigPath:     clusterctlConfigPath,
				KubeconfigPath:           bootstrapClusterProxy.GetKubeconfigPath(),
				InfrastructureProvider:   clusterctl.DefaultInfrastructureProvider,
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

		// Assert on byohost cluster label to match clusterName
		byoHostToBeReused = &infrastructurev1beta1.ByoHost{}
		Expect(bootstrapClusterProxy.GetClient().Get(ctx, byoHostLookupKey, byoHostToBeReused)).Should(Succeed())
		cluster, ok = byoHostToBeReused.Labels[clusterv1.ClusterLabelName]
		Expect(ok).To(BeTrue())
		Expect(cluster).To(Equal(clusterName))

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
