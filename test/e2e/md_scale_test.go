// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// nolint: testpackage
package e2e

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/docker/docker/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util"
)

var _ = Describe("When testing MachineDeployment scale out/in", func() {

	var (
		specName         = "md-scale"
		clusterResources *clusterctl.ApplyClusterTemplateAndWaitResult
	)

	BeforeEach(func() {

		clusterResources = new(clusterctl.ApplyClusterTemplateAndWaitResult)
	})

	It("Should successfully scale a MachineDeployment up and down upon changes to the MachineDeployment replica count", func() {
		clusterName := fmt.Sprintf("%s-%s", specName, util.RandomString(6))

		dClient, err := client.NewClientWithOpts(client.FromEnv)
		dockerClient = dClient
		Expect(err).NotTo(HaveOccurred())

		// TODO: Write agent logs to files for better debugging

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

		Expect(clusterResources.MachineDeployments[0].Spec.Replicas).To(Equal(pointer.Int32Ptr(2)))

	})

	JustAfterEach(func() {

	})

	AfterEach(func() {
	})
})
