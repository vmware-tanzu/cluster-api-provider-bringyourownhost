// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//nolint: testpackage
package e2e

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
	"os"
	"path/filepath"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util"
)

var _ = Describe("Cluster upgrade test [K8s-upgrade]", func() {

	var (
		ctx                          context.Context
		specName                     = "upgrade"
		namespace                    *corev1.Namespace
		cancelWatches                context.CancelFunc
		clusterResources             *clusterctl.ApplyClusterTemplateAndWaitResult
		byoHostCapacityPool          = 4
		byoHostName                  string
		dockerClient                 *client.Client
		allbyohostContainerIDs       []string
		allAgentLogFiles             []string
		kubernetesVersionUpgradeFrom = "v1.22.3"
		kubernetesVersionUpgradeTo   = "v1.23.5"
		etcdUpgradeVersion           = "3.5.1-0"
		coreDNSUpgradeVersion        = "v1.8.6"
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

	It("Should successfully upgrade cluster", func() {
		clusterName := fmt.Sprintf("%s-%s", specName, util.RandomString(6))
		var err error
		dockerClient, err = client.NewClientWithOpts(client.FromEnv)
		Expect(err).NotTo(HaveOccurred())

		By("Creating byohost capacity pool containing 4 docker hosts")
		for i := 0; i < byoHostCapacityPool; i++ {

			byoHostName = fmt.Sprintf("byohost-%s", util.RandomString(6))

			runner := ByoHostRunner{
				Context:               ctx,
				clusterConName:        clusterConName,
				ByoHostName:           byoHostName,
				Namespace:             namespace.Name,
				PathToHostAgentBinary: pathToHostAgentBinary,
				DockerClient:          dockerClient,
				NetworkInterface:      "kind",
				bootstrapClusterProxy: bootstrapClusterProxy,
				CommandArgs: map[string]string{
					"--bootstrap-kubeconfig": "/bootstrap.conf",
					"--namespace":            namespace.Name,
					"--v":                    "1",
				},
			}
			runner.BootstrapKubeconfigData = generateBootstrapKubeconfig(runner.Context, bootstrapClusterProxy, clusterConName)
			byohost, err := runner.SetupByoDockerHost()
			Expect(err).NotTo(HaveOccurred())
			output, byohostContainerID, err := runner.ExecByoDockerHost(byohost)
			allbyohostContainerIDs = append(allbyohostContainerIDs, byohostContainerID)
			Expect(err).NotTo(HaveOccurred())

			// read the log of host agent container in backend, and write it
			agentLogFile := fmt.Sprintf("/tmp/host-agent-%d.log", i)

			f := WriteDockerLog(output, agentLogFile)
			defer func() {
				deferredErr := f.Close()
				if deferredErr != nil {
					Showf("error closing file %s:, %v", agentLogFile, deferredErr)
				}
			}()
			allAgentLogFiles = append(allAgentLogFiles, agentLogFile)
		}

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
				KubernetesVersion:        kubernetesVersionUpgradeFrom,
				ControlPlaneMachineCount: pointer.Int64Ptr(1),
				WorkerMachineCount:       pointer.Int64Ptr(1),
			},
			WaitForClusterIntervals:      e2eConfig.GetIntervals(specName, "wait-cluster"),
			WaitForControlPlaneIntervals: e2eConfig.GetIntervals(specName, "wait-control-plane"),
			WaitForMachineDeployments:    e2eConfig.GetIntervals(specName, "wait-worker-nodes"),
		}, clusterResources)

		By("Upgrading the control plane")
		framework.UpgradeControlPlaneAndWaitForUpgrade(ctx, framework.UpgradeControlPlaneAndWaitForUpgradeInput{
			ClusterProxy:                bootstrapClusterProxy,
			Cluster:                     clusterResources.Cluster,
			ControlPlane:                clusterResources.ControlPlane,
			EtcdImageTag:                etcdUpgradeVersion,
			DNSImageTag:                 coreDNSUpgradeVersion,
			KubernetesUpgradeVersion:    kubernetesVersionUpgradeTo,
			WaitForMachinesToBeUpgraded: e2eConfig.GetIntervals(specName, "wait-machine-upgrade"),
			WaitForKubeProxyUpgrade:     e2eConfig.GetIntervals(specName, "wait-machine-upgrade"),
			WaitForDNSUpgrade:           e2eConfig.GetIntervals(specName, "wait-machine-upgrade"),
			WaitForEtcdUpgrade:          e2eConfig.GetIntervals(specName, "wait-machine-upgrade"),
		})

		By("Upgrading the machine deployment")
		framework.UpgradeMachineDeploymentsAndWait(ctx, framework.UpgradeMachineDeploymentsAndWaitInput{
			ClusterProxy:                bootstrapClusterProxy,
			Cluster:                     clusterResources.Cluster,
			UpgradeVersion:              kubernetesVersionUpgradeTo,
			MachineDeployments:          clusterResources.MachineDeployments,
			WaitForMachinesToBeUpgraded: e2eConfig.GetIntervals(specName, "wait-worker-nodes"),
		})

		By("Waiting until nodes are ready")
		workloadProxy := bootstrapClusterProxy.GetWorkloadCluster(ctx, namespace.Name, clusterResources.Cluster.Name)
		workloadClient := workloadProxy.GetClient()
		framework.WaitForNodesReady(ctx, framework.WaitForNodesReadyInput{
			Lister:            workloadClient,
			KubernetesVersion: kubernetesVersionUpgradeTo,
			Count:             int(clusterResources.ExpectedTotalNodes()),
			WaitForNodesReady: e2eConfig.GetIntervals(specName, "wait-nodes-ready"),
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

		if dockerClient != nil {
			for _, byohostContainerID := range allbyohostContainerIDs {
				err := dockerClient.ContainerStop(ctx, byohostContainerID, nil)
				Expect(err).NotTo(HaveOccurred())

				err = dockerClient.ContainerRemove(ctx, byohostContainerID, types.ContainerRemoveOptions{})
				Expect(err).NotTo(HaveOccurred())
			}

		}

		for _, agentLogFile := range allAgentLogFiles {
			err := os.Remove(agentLogFile)
			if err != nil {
				Showf("error removing file %s: %v", agentLogFile, err)
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
