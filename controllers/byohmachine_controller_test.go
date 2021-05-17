package controllers

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	infrastructurev1alpha3 "github.com/vmware-tanzu/cluster-api-provider-byoh/api/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

var _ = Describe("Controllers/ByohmachineController", func() {
	const (
		ByoMachineName      = "test-machine"
		ByoMachineNamespace = "default"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("create a new BYO machine", func() {
		It("should create a new BYO machine", func() {
			By("creating a new BYO machine")
			ctx := context.Background()
			byohMachine := &infrastructurev1alpha3.ByohMachine{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ByohMachine",
					APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha3",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      ByoMachineName,
					Namespace: ByoMachineNamespace,
				},
				Spec: infrastructurev1alpha3.ByohMachineSpec{
					Foo: "",
				},
			}
			Expect(k8sClient.Create(ctx, byohMachine)).Should(Succeed())
		})
	})
})
