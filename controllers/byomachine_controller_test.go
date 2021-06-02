package controllers

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/api/v1alpha4"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	byoMachineName      = "test-machine"
	byoHostName         = "test-host"
	byoMachineNamespace = "default"
)

var (
	ctx     context.Context
	byoHost *infrastructurev1alpha4.ByoHost
)

var _ = Describe("Controllers/ByomachineController", func() {
	Context("When a BYO Host is available", func() {
		BeforeEach(func() {
			ctx = context.Background()
			byoHost = &infrastructurev1alpha4.ByoHost{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ByoHost",
					APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      byoHostName,
					Namespace: byoMachineNamespace,
				},
				Spec: infrastructurev1alpha4.ByoHostSpec{},
			}
			Expect(k8sClient.Create(ctx, byoHost)).Should(Succeed())
		})

		It("claims the first available host", func() {
			byoMachine := &infrastructurev1alpha4.ByoMachine{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ByoMachine",
					APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      byoMachineName,
					Namespace: byoMachineNamespace,
				},
				Spec: infrastructurev1alpha4.ByoMachineSpec{},
			}
			Expect(k8sClient.Create(ctx, byoMachine)).Should(Succeed())

			byoHostLookupKey := types.NamespacedName{Name: byoHost.Name, Namespace: byoHost.Namespace}

			Eventually(func() *corev1.ObjectReference {
				createdByoHost := &infrastructurev1alpha4.ByoHost{}
				err := k8sClient.Get(ctx, byoHostLookupKey, createdByoHost)
				if err != nil {
					return nil
				}
				return createdByoHost.Status.MachineRef
			}).ShouldNot(BeNil())

		})
	})
})
