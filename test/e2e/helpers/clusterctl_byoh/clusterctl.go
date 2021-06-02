package clusterctl_byoh

import (
	"context"
	"fmt"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
)

func Logf(format string, a ...interface{}) {
	fmt.Fprintf(GinkgoWriter, "INFO: "+format+"\n", a...)
}

// TODO - Delete this method when we have implemented the complete e2e functionality. This is here only while we haven't completed creating the worker node.
func ApplyClusterTemplateAndWait(ctx context.Context, input clusterctl.ApplyClusterTemplateAndWaitInput, result *clusterctl.ApplyClusterTemplateAndWaitResult) {
	Expect(ctx).NotTo(BeNil(), "ctx is required for ApplyClusterTemplateAndWait")
	Expect(input.ClusterProxy).ToNot(BeNil(), "Invalid argument. input.ClusterProxy can't be nil when calling ApplyClusterTemplateAndWait")
	Expect(result).ToNot(BeNil(), "Invalid argument. result can't be nil when calling ApplyClusterTemplateAndWait")
	Expect(input.ConfigCluster.ControlPlaneMachineCount).ToNot(BeNil())
	Expect(input.ConfigCluster.WorkerMachineCount).ToNot(BeNil())

	Logf("Getting the cluster template yaml")
	workloadClusterTemplate := clusterctl.ConfigCluster(ctx, clusterctl.ConfigClusterInput{
		// pass reference to the management cluster hosting this test
		KubeconfigPath: input.ConfigCluster.KubeconfigPath,
		// pass the clusterctl config file that points to the local provider repository created for this test,
		ClusterctlConfigPath: input.ConfigCluster.ClusterctlConfigPath,
		// select templates
		Flavor: input.ConfigCluster.Flavor,
		// define template variables
		Namespace:                input.ConfigCluster.Namespace,
		ClusterName:              input.ConfigCluster.ClusterName,
		KubernetesVersion:        input.ConfigCluster.KubernetesVersion,
		ControlPlaneMachineCount: input.ConfigCluster.ControlPlaneMachineCount,
		WorkerMachineCount:       input.ConfigCluster.WorkerMachineCount,
		InfrastructureProvider:   input.ConfigCluster.InfrastructureProvider,
		// setup clusterctl logs folder
		LogFolder: input.ConfigCluster.LogFolder,
	})
	Expect(workloadClusterTemplate).ToNot(BeNil(), "Failed to get the cluster template")

	Logf("Applying the cluster template yaml to the cluster")
	Expect(input.ClusterProxy.Apply(ctx, workloadClusterTemplate, input.Args...)).To(Succeed())

	Logf("Waiting for the cluster infrastructure to be provisioned")
	result.Cluster = framework.DiscoveryAndWaitForCluster(ctx, framework.DiscoveryAndWaitForClusterInput{
		Getter:    input.ClusterProxy.GetClient(),
		Namespace: input.ConfigCluster.Namespace,
		Name:      input.ConfigCluster.ClusterName,
	}, input.WaitForClusterIntervals...)

	Logf("Waiting for control plane to be initialized")
	result.ControlPlane = framework.DiscoveryAndWaitForControlPlaneInitialized(ctx, framework.DiscoveryAndWaitForControlPlaneInitializedInput{
		Lister:  input.ClusterProxy.GetClient(),
		Cluster: result.Cluster,
	}, input.WaitForControlPlaneIntervals...)

	if input.CNIManifestPath != "" {
		Logf("Installing a CNI plugin to the workload cluster")
		workloadCluster := input.ClusterProxy.GetWorkloadCluster(ctx, result.Cluster.Namespace, result.Cluster.Name)

		cniYaml, err := os.ReadFile(input.CNIManifestPath)
		Expect(err).ShouldNot(HaveOccurred())

		Expect(workloadCluster.Apply(ctx, cniYaml)).ShouldNot(HaveOccurred())
	}

	Logf("Waiting for control plane to be ready")
	framework.WaitForControlPlaneAndMachinesReady(ctx, framework.WaitForControlPlaneAndMachinesReadyInput{
		GetLister:    input.ClusterProxy.GetClient(),
		Cluster:      result.Cluster,
		ControlPlane: result.ControlPlane,
	}, input.WaitForControlPlaneIntervals...)
}
