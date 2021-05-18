package controllers

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	infrastructurev1alpha3 "github.com/vmware-tanzu/cluster-api-provider-byoh/api/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterapi "sigs.k8s.io/cluster-api/api/v1alpha3"
	bootstrapv1 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1alpha3"
	"time"
)

var _ = Describe("Controllers/ByomachineController", func() {
	const (
		ByoHostName            = "test-host"
		ByoMachineNamespace       = "default"
		ByoMachineTemplateName    = "test-template"
		KubeAdmConfigTemplateName = "kubeadm-template"
		MachineDeploymentName     = "test-md"

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

			By("create a ByoMachineTemplate")
			ByoMachineTemplate := &infrastructurev1alpha3.ByoMachineTemplate{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ByoMachineTemplate",
					APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha3",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      ByoMachineTemplateName,
					Namespace: ByoMachineNamespace,
				},
				Spec: infrastructurev1alpha3.ByoMachineTemplateSpec{
					Foo: "Baz",
				},
			}
			Expect(k8sClient.Create(ctx, ByoMachineTemplate)).Should(Succeed())

			By("create a KubeAdmConfigTemplate")
			KubeAdmConfigTemplate := &bootstrapv1.KubeadmConfigTemplate{
				TypeMeta: metav1.TypeMeta{
					Kind:       "KubeAdmConfigTemplate",
					APIVersion: "bootstrap.cluster.x-k8s.io/v1alpha2",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      KubeAdmConfigTemplateName,
					Namespace: ByoMachineNamespace,
				},
				Spec: bootstrapv1.KubeadmConfigTemplateSpec{
					Template: bootstrapv1.KubeadmConfigTemplateResource{
						Spec: bootstrapv1.KubeadmConfigSpec{

						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, KubeAdmConfigTemplate)).Should(Succeed())

			By("create a MachineDeployment")
			MachineDeployment := &clusterapi.MachineDeployment{
				TypeMeta: metav1.TypeMeta{
					Kind:       "MachineDeployment",
					APIVersion: "cluster.x-k8s.io/v1alpha3",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      MachineDeploymentName,
					Namespace: ByoMachineNamespace,
				},
				Spec: clusterapi.MachineDeploymentSpec{
					ClusterName: "test",
					Template: clusterapi.MachineTemplateSpec{
						ObjectMeta: clusterapi.ObjectMeta{
							Name:      MachineDeploymentName,
							Namespace: ByoMachineNamespace,
						},
						Spec: clusterapi.MachineSpec{
							ClusterName: "test",
							Bootstrap: clusterapi.Bootstrap{
								ConfigRef: &corev1.ObjectReference{
									Kind:      "KubeAdmConfigTemplate",
									Namespace: ByoMachineNamespace,
									Name:      MachineDeploymentName,
								},
							},
							InfrastructureRef: corev1.ObjectReference{
								Kind:      "ByoMachineTemplate",
								Namespace: ByoMachineNamespace,
								Name:      MachineDeploymentName,
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, MachineDeployment)).Should(Succeed())
		})
	})
})
