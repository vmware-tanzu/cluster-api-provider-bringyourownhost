package main

import (
	"context"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/api/v1alpha4"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("HostAgent", func() {
	It("should register the BYOHost with the management cluster", func() {
		command := exec.Command(pathToHostAgentBinary, "--kubeconfig", kubeconfigFile.Name())
		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())
		Eventually(session, "60s").Should(gexec.Exit(0))

		byoHostLookupKey := types.NamespacedName{Name: "jaime.com", Namespace: "default"}
		Eventually(func() *infrastructurev1alpha4.ByoHost {
			createdByoHost := &infrastructurev1alpha4.ByoHost{}
			err := k8sClient.Get(context.Background(), byoHostLookupKey, createdByoHost)
			if err != nil {
				return nil
			}
			return createdByoHost
		}).ShouldNot(BeNil())
	})

	It("should error out if the host already exists", func() {
		ByoHost := &infrastructurev1alpha4.ByoHost{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ByoHost",
				APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      hostName,
				Namespace: namespace,
			},
			Spec: infrastructurev1alpha4.ByoHostSpec{},
		}
		err = k8sClient.Create(context.Background(), ByoHost)

		command := exec.Command(pathToHostAgentBinary, "--kubeconfig", kubeconfigFile.Name())
		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())

		Eventually(session).Should(gexec.Exit(255))
	})

	It("should retrun an error when invalid kubeconfig is passed in", func() {
		command := exec.Command(pathToHostAgentBinary, "--kubeconfig", "non-existent-path")
		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())
		Eventually(session).Should(gexec.Exit(255))
	})
})
