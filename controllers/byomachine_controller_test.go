package controllers

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	infrastructurev1alpha3 "github.com/vmware-tanzu/cluster-api-provider-byoh/api/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"
)

var _ = Describe("Controllers/ByomachineController", func() {
	const (
		ByoMachineName      = "test-machine"
		ByoHostName         = "test-host"
		ByoMachineNamespace = "default"

		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("create all base CRDs", func() {
		It("create all CRDs", func() {

			ctx := context.Background()

			By("create a ByoHost")
			ByoHost := &infrastructurev1alpha3.ByoHost{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ByoHost",
					APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha3",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      ByoHostName,
					Namespace: ByoMachineNamespace,
				},
				Spec: infrastructurev1alpha3.ByoHostSpec{
					Foo: "Baz",
				},
			}
			Expect(k8sClient.Create(ctx, ByoHost)).Should(Succeed())

			ByoMachine := &infrastructurev1alpha3.ByoMachine{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ByoMachine",
					APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha3",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      ByoMachineName,
					Namespace: ByoMachineNamespace,
				},
				Spec: infrastructurev1alpha3.ByoMachineSpec{
					Foo: "bar",
				},
			}
			Expect(k8sClient.Create(ctx, ByoMachine)).Should(Succeed())

			By("fetching the Byomachine")
			ByoMachineLookupKey := types.NamespacedName{Name: ByoMachineName, Namespace: ByoMachineNamespace}
			createdByoMachine := &infrastructurev1alpha3.ByoMachine{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, ByoMachineLookupKey, createdByoMachine)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			Expect(createdByoMachine.Spec.Foo).Should(Equal("bar"))

			ByoHostLookupKey := types.NamespacedName{Name: ByoHostName, Namespace: ByoMachineNamespace}

			Eventually(func() *corev1.ObjectReference {
				createdByoHost := &infrastructurev1alpha3.ByoHost{}
				err := k8sClient.Get(ctx, ByoHostLookupKey, createdByoHost)
				if err != nil {
					return nil
				}
				return createdByoHost.Status.MachineRef
			}).ShouldNot(BeNil())

		})
	})
})
