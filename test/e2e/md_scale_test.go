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
	"k8s.io/utils/pointer"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util"
)

var _ = Describe("When testing MachineDeployment scale out/in [Before-PR-Merging]", func() {

	var (
		specName               = "md-scale"
		dockerClient           *client.Client
		err                    error
		byoHostCapacityPool    = 6
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
		for i := 0; i < byoHostCapacityPool; i++ {
			byoHostName := fmt.Sprintf("byohost-%s", util.RandomString(6))
			runner.ByoHostName = byoHostName

			byohost, err := runner.SetupByoDockerHost()
			Expect(err).NotTo(HaveOccurred())

			output, byohostContainerID, err := runner.ExecByoDockerHost(byohost)
			Expect(err).NotTo(HaveOccurred())
			allContainerOutputs = append(allContainerOutputs, output)
			allbyohostContainerIDs = append(allbyohostContainerIDs, byohostContainerID)

			// read the log of host agent container in backend, and write it
			agentLogFile := fmt.Sprintf("/tmp/host-agent-%d.log", i)
			f := WriteDockerLog(output, agentLogFile)
			allAgentLogFiles = append(allAgentLogFiles, f)
		}
	})

	It("Should successfully scale a MachineDeployment up and down upon changes to the MachineDeployment replica count", func() {
		clusterName := fmt.Sprintf("%s-%s", specName, util.RandomString(6))
		By("creating a workload cluster with one control plane node and one worker node")
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
				ControlPlaneMachineCount: pointer.Int64Ptr(3),
				WorkerMachineCount:       pointer.Int64Ptr(1),
			},
			WaitForClusterIntervals:      e2eConfig.GetIntervals(specName, "wait-cluster"),
			WaitForControlPlaneIntervals: e2eConfig.GetIntervals(specName, "wait-control-plane"),
			WaitForMachineDeployments:    e2eConfig.GetIntervals(specName, "wait-worker-nodes"),
		}, clusterResources)

		Expect(clusterResources.MachineDeployments[0].Spec.Replicas).To(Equal(pointer.Int32Ptr(1)))

		By("Scaling the MachineDeployment out to 3")
		framework.ScaleAndWaitMachineDeployment(ctx, framework.ScaleAndWaitMachineDeploymentInput{
			ClusterProxy:              bootstrapClusterProxy,
			Cluster:                   clusterResources.Cluster,
			MachineDeployment:         clusterResources.MachineDeployments[0],
			Replicas:                  3,
			WaitForMachineDeployments: e2eConfig.GetIntervals(specName, "wait-worker-nodes"),
		})

		Expect(clusterResources.MachineDeployments[0].Spec.Replicas).To(Equal(pointer.Int32Ptr(3)))

		By("Scaling the MachineDeployment down to 2")
		framework.ScaleAndWaitMachineDeployment(ctx, framework.ScaleAndWaitMachineDeploymentInput{
			ClusterProxy:              bootstrapClusterProxy,
			Cluster:                   clusterResources.Cluster,
			MachineDeployment:         clusterResources.MachineDeployments[0],
			Replicas:                  2,
			WaitForMachineDeployments: e2eConfig.GetIntervals(specName, "wait-worker-nodes"),
		})
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

		if dockerClient != nil {
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
			Showf("error removing file %s: %v", ReadByohControllerManagerLogShellFile, err)
		}
		err = os.Remove(ReadAllPodsShellFile)
		if err != nil {
			Showf("error removing file %s: %v", ReadAllPodsShellFile, err)
		}
	})
})
