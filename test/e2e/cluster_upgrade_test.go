// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// nolint: testpackage
package e2e

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"path/filepath"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/utils/pointer"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"os"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	bootstrapv1 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1beta1"
	containerutil "sigs.k8s.io/cluster-api/util/container"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"strconv"
	"strings"
	"time"
)

var _ = Describe("Cluster upgrade test [K8s-upgrade]", func() {

	var (
		ctx                    context.Context
		specName               = "cu"
		namespace              *corev1.Namespace
		cancelWatches          context.CancelFunc
		clusterResources       *clusterctl.ApplyClusterTemplateAndWaitResult
		dockerClient           *client.Client
		byoHostCapacityPool    = 4
		byoHostName            string
		allbyohostContainerIDs []string
		allAgentLogFiles       []string
	)

	ctx = context.TODO()


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

	It("Should successfully scale a MachineDeployment up and down upon changes to the MachineDeployment replica count", func() {
		clusterName := fmt.Sprintf("%s-%s", specName, util.RandomString(6))

		dClient, err := client.NewClientWithOpts(client.FromEnv)
		dockerClient = dClient
		Expect(err).NotTo(HaveOccurred())

		By("Creating byohost capacity pool containing 4 hosts")
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
			By("Created host")
		}
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
				KubernetesVersion:        "v1.23.4",
				ControlPlaneMachineCount: pointer.Int64Ptr(1),
				WorkerMachineCount:       pointer.Int64Ptr(1),
			},
			WaitForClusterIntervals:      e2eConfig.GetIntervals(specName, "wait-cluster"),
			WaitForControlPlaneIntervals: e2eConfig.GetIntervals(specName, "wait-control-plane"),
			WaitForMachineDeployments:    e2eConfig.GetIntervals(specName, "wait-worker-nodes"),
		}, clusterResources)

		//Expect(clusterResources.MachineDeployments[0].Spec.Replicas).To(Equal(pointer.Int32Ptr(1)))

		By("Upgrading the KCP")

		// hack start
		mgmtClient := bootstrapClusterProxy.GetClient()

		By("Patching the new kubernetes version to KCP")
		patchHelper, err := patch.NewHelper(clusterResources.ControlPlane, mgmtClient)
		Expect(err).ToNot(HaveOccurred())

		clusterResources.ControlPlane.Spec.Version = "v1.23.5"
		//if input.UpgradeMachineTemplate != nil {
		//	clusterResources.ControlPlane.Spec.MachineTemplate.InfrastructureRef.Name = *input.UpgradeMachineTemplate
		//}
		// If the ClusterConfiguration is not specified, create an empty one.
		if clusterResources.ControlPlane.Spec.KubeadmConfigSpec.ClusterConfiguration == nil {
			clusterResources.ControlPlane.Spec.KubeadmConfigSpec.ClusterConfiguration = new(bootstrapv1.ClusterConfiguration)
		}

		if clusterResources.ControlPlane.Spec.KubeadmConfigSpec.ClusterConfiguration.Etcd.Local == nil {
			clusterResources.ControlPlane.Spec.KubeadmConfigSpec.ClusterConfiguration.Etcd.Local = new(bootstrapv1.LocalEtcd)
		}

		//clusterResources.ControlPlane.Spec.KubeadmConfigSpec.ClusterConfiguration.Etcd.Local.ImageMeta.ImageTag = e2eConfig.GetVariable("ETCD_VERSION_UPGRADE_TO")
		//clusterResources.ControlPlane.Spec.KubeadmConfigSpec.ClusterConfiguration.DNS.ImageMeta.ImageTag = e2eConfig.GetVariable("COREDNS_VERSION_UPGRADE_TO")
		clusterResources.ControlPlane.Spec.KubeadmConfigSpec.ClusterConfiguration.Etcd.Local.ImageMeta.ImageTag = ""
		clusterResources.ControlPlane.Spec.KubeadmConfigSpec.ClusterConfiguration.DNS.ImageMeta.ImageTag = ""

		Eventually(func() error {
			return patchHelper.Patch(ctx, clusterResources.ControlPlane)
		}, 1 * time.Minute, 3 * time.Second).Should(Succeed())

		By("Waiting for control-plane machines to have the upgraded kubernetes version")
		//WaitForControlPlaneMachinesToBeUpgraded(ctx, WaitForControlPlaneMachinesToBeUpgradedInput{
		//	Lister:                   mgmtClient,
		//	Cluster:                  input.Cluster,
		//	MachineCount:             int(*input.ControlPlane.Spec.Replicas),
		//	KubernetesUpgradeVersion: input.KubernetesUpgradeVersion,
		//}, input.WaitForMachinesToBeUpgraded...)

		Eventually(func() (int, error) {
			machines := framework.GetControlPlaneMachinesByCluster(ctx, framework.GetControlPlaneMachinesByClusterInput{
				Lister:      mgmtClient,
				ClusterName: clusterResources.Cluster.Name,
				Namespace:   clusterResources.Cluster.Namespace,
			})

			upgraded := 0
			for _, machine := range machines {
				m := machine
				if *m.Spec.Version == "v1.23.5" && conditions.IsTrue(&m, clusterv1.MachineNodeHealthyCondition) {
					upgraded++
				}
			}
			By("len(machines)")
			By(strconv.Itoa(len(machines)))
			By("upgraded")
			By(strconv.Itoa(upgraded))

			if len(machines) > upgraded {
				return 0, errors.New("old nodes remain")
			}
			return upgraded, nil
		}, e2eConfig.GetIntervals(specName, "wait-machine-upgrade")...).Should(Equal(int(*clusterResources.ControlPlane.Spec.Replicas)))


		By("Waiting for kube-proxy to have the upgraded kubernetes version")
		workloadCluster := bootstrapClusterProxy.GetWorkloadCluster(ctx, clusterResources.Cluster.Namespace, clusterResources.Cluster.Name)
		workloadClient := workloadCluster.GetClient()


		By("Ensuring kube-proxy has the correct image")

		Eventually(func() (bool, error) {
			ds := &appsv1.DaemonSet{}

			if err := workloadClient.Get(ctx, ctrlclient.ObjectKey{Name: "kube-proxy", Namespace: metav1.NamespaceSystem}, ds); err != nil {
				return false, err
			}
			By("Actual")
			By(ds.Spec.Template.Spec.Containers[0].Image)
			By("Desired")
			By(containerutil.SemverToOCIImageTag("v1.23.5"))
			if ds.Spec.Template.Spec.Containers[0].Image == "k8s.gcr.io/kube-proxy:"+containerutil.SemverToOCIImageTag("v1.23.5") {
				return true, nil
			}
			return false, nil
		}, e2eConfig.GetIntervals(specName, "wait-machine-upgrade")...).Should(BeTrue())

		//WaitForKubeProxyUpgrade(ctx, WaitForKubeProxyUpgradeInput{
		//	Getter:            workloadClient,
		//	KubernetesVersion: input.KubernetesUpgradeVersion,
		//}, input.WaitForKubeProxyUpgrade...)

		By("Waiting for CoreDNS to have the upgraded image tag")
		By("Ensuring CoreDNS has the correct image")

		Eventually(func() (bool, error) {
			d := &appsv1.Deployment{}

			if err := workloadClient.Get(ctx, ctrlclient.ObjectKey{Name: "coredns", Namespace: metav1.NamespaceSystem}, d); err != nil {
				return false, err
			}

			// NOTE: coredns image name has changed over time (k8s.gcr.io/coredns,
			// k8s.gcr.io/coredns/coredns), so we are checking only if the version actually changed.
			By("Actual")
			By(d.Spec.Template.Spec.Containers[0].Image)
			By("Desired")
			By(e2eConfig.GetVariable("COREDNS_VERSION_UPGRADE_TO"))
			if strings.HasSuffix(d.Spec.Template.Spec.Containers[0].Image, fmt.Sprintf(":%s", e2eConfig.GetVariable("COREDNS_VERSION_UPGRADE_TO"))) {
				return true, nil
			}
			return false, nil
		}, e2eConfig.GetIntervals(specName, "wait-machine-upgrade")...).Should(BeTrue())

		//WaitForDNSUpgrade(ctx, WaitForDNSUpgradeInput{
		//	Getter:     workloadClient,
		//	DNSVersion: input.DNSImageTag,
		//}, input.WaitForDNSUpgrade...)

		By("Waiting for etcd to have the upgraded image tag")
		lblSelector, err := labels.Parse("component=etcd")
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() (bool, error) {
			podList := &corev1.PodList{}
			if err := workloadClient.List(ctx, podList, &ctrlclient.ListOptions{LabelSelector: lblSelector}); err != nil {
				return false, err
			}
			By("Desired")
			By(e2eConfig.GetVariable("ETCD_VERSION_UPGRADE_TO"))
			// all pods in the list should satisfy the condition
			err := framework.EtcdImageTagCondition(e2eConfig.GetVariable("ETCD_VERSION_UPGRADE_TO"), int(*clusterResources.ControlPlane.Spec.Replicas))(podList)
			if err != nil {
				By(err.Error())
				return false, err
			}
			return true, nil
		}, e2eConfig.GetIntervals(specName, "wait-machine-upgrade")...).Should(BeTrue())

		//lblSelector, err := labels.Parse("component=etcd")
		//Expect(err).ToNot(HaveOccurred())
		//WaitForPodListCondition(ctx, WaitForPodListConditionInput{
		//	Lister:      workloadClient,
		//	ListOptions: &client.ListOptions{LabelSelector: lblSelector},
		//	Condition:   EtcdImageTagCondition(input.EtcdImageTag, int(*input.ControlPlane.Spec.Replicas)),
		//}, input.WaitForEtcdUpgrade...)

		// hack end


		//framework.UpgradeControlPlaneAndWaitForUpgrade(ctx, framework.UpgradeControlPlaneAndWaitForUpgradeInput{
		//	ClusterProxy:                bootstrapClusterProxy,
		//	Cluster:                     clusterResources.Cluster,
		//	ControlPlane:                clusterResources.ControlPlane,
		//	EtcdImageTag:                e2eConfig.GetVariable("ETCD_VERSION_UPGRADE_TO"),
		//	DNSImageTag:                 e2eConfig.GetVariable("COREDNS_VERSION_UPGRADE_TO"),
		//	KubernetesUpgradeVersion:    "v1.23.5",
		//	//UpgradeMachineTemplate:      upgradeCPMachineTemplateTo,
		//	WaitForMachinesToBeUpgraded: e2eConfig.GetIntervals(specName, "wait-machine-upgrade"),
		//	WaitForKubeProxyUpgrade:     e2eConfig.GetIntervals(specName, "wait-machine-upgrade"),
		//	WaitForDNSUpgrade:           e2eConfig.GetIntervals(specName, "wait-machine-upgrade"),
		//	WaitForEtcdUpgrade:          e2eConfig.GetIntervals(specName, "wait-machine-upgrade"),
		//})
		By("Upgrading the machine deployment")
		framework.UpgradeMachineDeploymentsAndWait(ctx, framework.UpgradeMachineDeploymentsAndWaitInput{
			ClusterProxy:                bootstrapClusterProxy,
			Cluster:                     clusterResources.Cluster,
			UpgradeVersion:              "v1.23.5",
			//UpgradeMachineTemplate:      upgradeWorkersMachineTemplateTo,
			MachineDeployments:          clusterResources.MachineDeployments,
			WaitForMachinesToBeUpgraded: e2eConfig.GetIntervals(specName, "wait-worker-nodes"),
		})

		By("Waiting until nodes are ready")
		//workloadProxy := bootstrapClusterProxy.GetWorkloadCluster(ctx, namespace.Name, clusterResources.Cluster.Name)
		//workloadClient := workloadProxy.GetClient()
		framework.WaitForNodesReady(ctx, framework.WaitForNodesReadyInput{
			Lister:            workloadClient,
			KubernetesVersion:"v1.23.5",
			Count:             int(clusterResources.ExpectedTotalNodes()),
			WaitForNodesReady: e2eConfig.GetIntervals(specName, "wait-nodes-ready"),
		})
		//capi_e2e.ClusterUpgradeConformanceSpec(ctx, func() capi_e2e.ClusterUpgradeConformanceSpecInput {
		//	return capi_e2e.ClusterUpgradeConformanceSpecInput{
		//		E2EConfig:                e2eConfig,
		//		ClusterctlConfigPath:     clusterctlConfigPath,
		//		BootstrapClusterProxy:    bootstrapClusterProxy,
		//		ArtifactFolder:           artifactFolder,
		//		SkipCleanup:              skipCleanup,
		//		SkipConformanceTests:     true,
		//		//ControlPlaneMachineCount: pointer.Int64(1),
		//		//WorkerMachineCount:       pointer.Int64(1),
		//		//Flavor:                   pointer.String(clusterctl.DefaultFlavor),
		//	}
		//})
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
