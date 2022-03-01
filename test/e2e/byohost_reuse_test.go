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
	infrastructurev1beta1 "github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/apis/infrastructure/v1beta1"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util"
)

var _ = Describe("When BYO Host rejoins the capacity pool", func() {

	var (
		specName = "byohost-reuse"
	)

	It("Should reuse the same BYO Host after it is reset", func() {

		clusterResources := new(clusterctl.ApplyClusterTemplateAndWaitResult)

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
})

func setDockerClient(dc *client.Client) {
	dockerClient = dc
}

func getDockerClient() *client.Client {
	return dockerClient
}
