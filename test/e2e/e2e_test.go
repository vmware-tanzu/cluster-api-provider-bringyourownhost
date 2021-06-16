/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/api/v1alpha4"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/test/e2e/helpers/clusterctl_byoh"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
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
var _ = Describe("When BYOH joins existing cluster", func() {

	var ctx context.Context
	var (
		specName         = "quick-start"
		namespace        *corev1.Namespace
		cancelWatches    context.CancelFunc
		clusterResources *clusterctl.ApplyClusterTemplateAndWaitResult
	)

	BeforeEach(func() {

		ctx = context.TODO()
		Expect(ctx).NotTo(BeNil(), "ctx is required for %s spec", specName)

		Expect(e2eConfig).ToNot(BeNil(), "Invalid argument. e2eConfig can't be nil when calling %s spec", specName)
		Expect(clusterctlConfigPath).To(BeAnExistingFile(), "Invalid argument. clusterctlConfigPath must be an existing file when calling %s spec", specName)
		Expect(bootstrapClusterProxy).ToNot(BeNil(), "Invalid argument. bootstrapClusterProxy can't be nil when calling %s spec", specName)
		Expect(os.MkdirAll(artifactFolder, 0755)).To(Succeed(), "Invalid argument. artifactFolder can't be created for %s spec", specName)

		Expect(e2eConfig.Variables).To(HaveKey(KubernetesVersion))

		// Setup a Namespace where to host objects for this spec and create a watcher for the namespace events.
		namespace, cancelWatches = setupSpecNamespace(ctx, specName, bootstrapClusterProxy, artifactFolder)
		clusterResources = new(clusterctl.ApplyClusterTemplateAndWaitResult)
	})

	It("Should create a workload cluster with single BYOH host", func() {

		clusterName := fmt.Sprintf("%s-%s", specName, util.RandomString(6))
		// hostName := "test.com"

		// ByoHost := &infrastructurev1alpha4.ByoHost{
		// 	TypeMeta: metav1.TypeMeta{
		// 		Kind:       "ByoHost",
		// 		APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
		// 	},
		// 	ObjectMeta: metav1.ObjectMeta{
		// 		Name:      hostName,
		// 		Namespace: namespace.Name,
		// 	},
		// 	Spec: infrastructurev1alpha4.ByoHostSpec{},
		// }
		client := bootstrapClusterProxy.GetClient()
		// Expect(client.Create(ctx, ByoHost)).Should(Succeed())

		clusterctl_byoh.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
			ClusterProxy: bootstrapClusterProxy,
			ConfigCluster: clusterctl.ConfigClusterInput{
				LogFolder:                filepath.Join(artifactFolder, "clusters", bootstrapClusterProxy.GetName()),
				ClusterctlConfigPath:     clusterctlConfigPath,
				KubeconfigPath:           bootstrapClusterProxy.GetKubeconfigPath(),
				InfrastructureProvider:   "byoh:v0.4.0",
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

		ByoHostLookupKey := types.NamespacedName{Name: "jaime.com", Namespace: "default"}

		// TODO: Remove the below interim assertion after implementing Host Agent

		Eventually(func() *corev1.ObjectReference {
			createdByoHost := &infrastructurev1alpha4.ByoHost{}
			err := client.Get(ctx, ByoHostLookupKey, createdByoHost)
			if err != nil {
				return nil
			}
			return createdByoHost.Status.MachineRef
		}).ShouldNot(BeNil())

	})

	AfterEach(func() {
		// Dumps all the resources in the spec namespace, then cleanups the cluster object and the spec namespace itself.
		dumpSpecResourcesAndCleanup(ctx, specName, bootstrapClusterProxy, artifactFolder, namespace, cancelWatches, clusterResources.Cluster, e2eConfig.GetIntervals, skipCleanup)
	})
})
