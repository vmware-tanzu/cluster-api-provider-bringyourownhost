// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// nolint: testpackage
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
	infrastructurev1beta1 "github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/apis/infrastructure/v1beta1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util"
)

var _ = Describe("When BYO Host rejoins the capacity pool [Before-PR-Merging]", func() {

	var (
		specName               = "byohost-reuse"
		dockerClient           *client.Client
		err                    error
		byoHostName1           = "byohost-1"
		byoHostName2           = "byohost-for-reuse"
		allbyohostContainerIDs []string
		allAgentLogFiles       []*os.File
		allContainerOutputs    []types.HijackedResponse
	)

	BeforeEach(func() {
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
				"--kubeconfig": "/mgmt.conf",
				"--namespace":  namespace.Name,
				"--v":          "1",
			},
		}

		By("Creating byohost capacity pool containing 6 hosts")
		for i := 0; i < 2; i++ {
			agentLogFile := ""
			if i == 0 {
				runner.ByoHostName = byoHostName1
				agentLogFile = "/tmp/host-agent1.log"
			} else {
				runner.ByoHostName = byoHostName2
				agentLogFile = "/tmp/host-agent-reuse.log"
			}

			byohost, err := runner.SetupByoDockerHost()
			Expect(err).NotTo(HaveOccurred())

			output, byohostContainerID, err := runner.ExecByoDockerHost(byohost)
			Expect(err).NotTo(HaveOccurred())
			allContainerOutputs = append(allContainerOutputs, output)
			allbyohostContainerIDs = append(allbyohostContainerIDs, byohostContainerID)

			f := WriteDockerLog(output, agentLogFile)
			allAgentLogFiles = append(allAgentLogFiles, f)
		}

	})

	It("Should reuse the same BYO Host after it is reset", func() {
		clusterName := fmt.Sprintf("%s-%s", specName, util.RandomString(6))

		By("Creating a cluster")
		setControlPlaneIP(context.Background(), dockerClient)
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
			ShowInfo(allAgentLogFiles)
		}
	})

	AfterEach(func() {
		// Dumps all the resources in the spec namespace, then cleanups the cluster object and the spec namespace itself.
		dumpSpecResourcesAndCleanup(ctx, specName, bootstrapClusterProxy, artifactFolder, namespace, cancelWatches, clusterResources.Cluster, e2eConfig.GetIntervals, skipCleanup)

		for _, output := range allContainerOutputs {
			output.Close()
		}

		if dockerClient != nil && len(allbyohostContainerIDs) != 0 {
			for _, byohostContainerID := range allbyohostContainerIDs {
				err := dockerClient.ContainerStop(ctx, byohostContainerID, nil)
				Expect(err).NotTo(HaveOccurred())

				err = dockerClient.ContainerRemove(ctx, byohostContainerID, types.ContainerRemoveOptions{})
				Expect(err).NotTo(HaveOccurred())
			}
		}

		for _, f := range allAgentLogFiles {
			if err := f.Close(); err != nil {
				Showf("error closing file %s:, %v", f.Name(), err)
			}

			if err := os.Remove(f.Name()); err != nil {
				Showf("error removing file %s: %v", f.Name(), err)
			}
		}
		err := os.Remove(ReadByohControllerManagerLogShellFile)
		if err != nil {
			Showf("Failed to remove file %s: %v", ReadByohControllerManagerLogShellFile, err)
		}
		err = os.Remove(ReadAllPodsShellFile)
		if err != nil {
			Showf("Failed to remove file %s: %v", ReadAllPodsShellFile, err)
		}
	})
})
