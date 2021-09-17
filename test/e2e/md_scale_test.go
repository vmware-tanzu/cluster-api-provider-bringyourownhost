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
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util"
)

var _ = Describe("When testing MachineDeployment scale out/in", func() {

	var (
		ctx                    context.Context
		specName               = "md-scale"
		namespace              *corev1.Namespace
		cancelWatches          context.CancelFunc
		clusterResources       *clusterctl.ApplyClusterTemplateAndWaitResult
		dockerClient           *client.Client
		err                    error
		byoHostCapacityPool    = 5
		byoHostName            string
		allbyohostContainerIDs []string
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

	It("Should successfully scale a MachineDeployment up and down upon changes to the MachineDeployment replica count", func() {
		clusterName := fmt.Sprintf("%s-%s", specName, util.RandomString(6))

		dockerClient, err = client.NewClientWithOpts(client.FromEnv)
		Expect(err).NotTo(HaveOccurred())

		By("Creating byohost capacity pool containing 5 hosts")
		for i := 0; i < byoHostCapacityPool; i++ {
			byoHostName = fmt.Sprintf("byohost-%s", util.RandomString(6))
			_, byohostContainerID, err := setupByoDockerHost(ctx, clusterConName, byoHostName, namespace.Name, dockerClient, bootstrapClusterProxy)
			allbyohostContainerIDs = append(allbyohostContainerIDs, byohostContainerID)
			Expect(err).NotTo(HaveOccurred())
		}

		// TODO: Write agent logs to files for better debugging

		By("creating a workload cluster with one control plane node and one worker node")
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

		Expect(clusterResources.MachineDeployments[0].Spec.Replicas).To(Equal(pointer.Int32Ptr(1)))

		By("Scaling the MachineDeployment out to 5")
		framework.ScaleAndWaitMachineDeployment(ctx, framework.ScaleAndWaitMachineDeploymentInput{
			ClusterProxy:              bootstrapClusterProxy,
			Cluster:                   clusterResources.Cluster,
			MachineDeployment:         clusterResources.MachineDeployments[0],
			Replicas:                  5,
			WaitForMachineDeployments: e2eConfig.GetIntervals(specName, "wait-worker-nodes"),
		})

		Expect(clusterResources.MachineDeployments[0].Spec.Replicas).To(Equal(pointer.Int32Ptr(5)))

		By("Scaling the MachineDeployment down to 3")
		framework.ScaleAndWaitMachineDeployment(ctx, framework.ScaleAndWaitMachineDeploymentInput{
			ClusterProxy:              bootstrapClusterProxy,
			Cluster:                   clusterResources.Cluster,
			MachineDeployment:         clusterResources.MachineDeployments[0],
			Replicas:                  3,
			WaitForMachineDeployments: e2eConfig.GetIntervals(specName, "wait-worker-nodes"),
		})

		Expect(clusterResources.MachineDeployments[0].Spec.Replicas).To(Equal(pointer.Int32Ptr(3)))

	})

	JustAfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			ShowInfo()
		}
	})

	AfterEach(func() {
		if dockerClient != nil {
			for _, byohostContainerID := range allbyohostContainerIDs {
				err := dockerClient.ContainerStop(ctx, byohostContainerID, nil)
				Expect(err).NotTo(HaveOccurred())

				err = dockerClient.ContainerRemove(ctx, byohostContainerID, types.ContainerRemoveOptions{})
				Expect(err).NotTo(HaveOccurred())
			}

		}

		os.Remove(AgentLogFile)
		os.Remove(ReadByohControllerManagerLogShellFile)
		os.Remove(ReadAllPodsShellFile)

		// Dumps all the resources in the spec namespace, then cleanups the cluster object and the spec namespace itself.
		dumpSpecResourcesAndCleanup(ctx, specName, bootstrapClusterProxy, artifactFolder, namespace, cancelWatches, clusterResources.Cluster, e2eConfig.GetIntervals, skipCleanup)
	})
})
