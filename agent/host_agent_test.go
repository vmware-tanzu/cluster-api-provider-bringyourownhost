package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/api/v1alpha4"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/cluster-api/util/patch"
)

var _ = Describe("Agent", func() {

	Context("negative case", func() {
		var (
			ns              *corev1.Namespace
			err             error
			hostName        string
			fakedKubeConfig string = "fake-kubeconfig-path"
			session         *gexec.Session
		)

		BeforeEach(func() {
			ns, err = createNamespace("testns-" + RandStr(5))
			Expect(err).NotTo(HaveOccurred(), "failed to create test namespace")

			hostName, err = os.Hostname()
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			err = k8sClient.Delete(context.TODO(), ns)
			Expect(err).NotTo(HaveOccurred(), "failed to delete test namespace")
			session.Terminate().Wait()
		})

		It("should error out if the host already exists", func() {
			_, err := createByoHost(hostName, ns.Name)
			Expect(err).ToNot(HaveOccurred())

			command := exec.Command(pathToHostAgentBinary, "--kubeconfig", kubeconfigFile.Name(), "--namespace", ns.Name)
			session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ToNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(255))
		})

		It("should return an error when invalid kubeconfig is passed in", func() {
			command := exec.Command(pathToHostAgentBinary, "--kubeconfig", fakedKubeConfig)
			session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ToNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(255))
		})
	})

	Context("When the host agent is able to connect to API Server", func() {

		var (
			ns       *corev1.Namespace
			session  *gexec.Session
			err      error
			dir      string
			hostName string
		)

		BeforeEach(func() {
			ns, err = createNamespace("testns-" + RandStr(5))
			Expect(err).NotTo(HaveOccurred(), "failed to create test namespace")

			hostName, err = os.Hostname()
			Expect(err).NotTo(HaveOccurred())

			command := exec.Command(pathToHostAgentBinary, "--kubeconfig", kubeconfigFile.Name(), "--namespace", ns.Name)

			session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ToNot(HaveOccurred())

			dir, err = ioutil.TempDir("", "cloudinit")
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			err = k8sClient.Delete(context.TODO(), ns)
			Expect(err).ToNot(HaveOccurred())
			os.RemoveAll(dir)
			session.Terminate().Wait()
		})

		It("should register the BYOHost with the management cluster", func() {
			byoHostLookupKey := types.NamespacedName{Name: hostName, Namespace: ns.Name}
			Eventually(func() *infrastructurev1alpha4.ByoHost {
				createdByoHost := &infrastructurev1alpha4.ByoHost{}
				err := k8sClient.Get(context.TODO(), byoHostLookupKey, createdByoHost)
				if err != nil {
					return nil
				}
				return createdByoHost
			}).ShouldNot(BeNil())
		})

		It("should bootstrap the node when MachineRef is set", func() {
			byoHost := &infrastructurev1alpha4.ByoHost{}
			byoHostLookupKey := types.NamespacedName{Name: hostName, Namespace: ns.Name}
			Eventually(func() bool {
				err = k8sClient.Get(context.TODO(), byoHostLookupKey, byoHost)
				return err == nil
			}).ShouldNot(BeFalse())

			fileToCreate := path.Join(dir, "test-directory", "test-file.txt")

			bootstrapSecretUnencoded := fmt.Sprintf(`write_files:
- path: %s
  content: expected-content
runCmd:
- echo -n ' run cmd' >> %s`, fileToCreate, fileToCreate)

			secret, err := createSecret("bootstrap-secret-1", bootstrapSecretUnencoded, ns.Name)
			Expect(err).ToNot(HaveOccurred())

			machine, err := createMachine(&secret.Name, "test-machine", ns.Name)
			Expect(err).ToNot(HaveOccurred())

			byoMachine, err := createByoMachine("test-byomachine", ns.Name, machine)
			Expect(err).ToNot(HaveOccurred())

			helper, err := patch.NewHelper(byoHost, k8sClient)
			Expect(err).ToNot(HaveOccurred())

			byoHost.Status.MachineRef = &corev1.ObjectReference{
				Kind:       "ByoMachine",
				Namespace:  ns.Name,
				Name:       byoMachine.Name,
				UID:        byoMachine.UID,
				APIVersion: byoHost.APIVersion,
			}
			err = helper.Patch(context.TODO(), byoHost)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() corev1.ConditionStatus {
				createdByoHost := &infrastructurev1alpha4.ByoHost{}
				err := k8sClient.Get(context.TODO(), byoHostLookupKey, createdByoHost)
				if err != nil {
					return corev1.ConditionFalse
				}
				for _, condition := range createdByoHost.Status.Conditions {
					if condition.Type == infrastructurev1alpha4.K8sComponentsInstalledCondition {
						return condition.Status
					}
				}
				return corev1.ConditionFalse
			}).Should(Equal(corev1.ConditionTrue))

			Eventually(func() string {
				buffer, err := ioutil.ReadFile(fileToCreate)
				if err != nil {
					return ""
				}
				return string(buffer)
			}).Should(Equal("expected-content run cmd"))

		})
	})

})

func RandStr(length int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyz"
	bytes := []byte(str)
	result := []byte{}
	rand.Seed(time.Now().UnixNano() + int64(rand.Intn(100)))
	for i := 0; i < length; i++ {
		result = append(result, bytes[rand.Intn(len(bytes))])
	}
	return string(result)
}

func createNamespace(namespace string) (*corev1.Namespace, error) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: namespace},
	}
	err := k8sClient.Create(context.TODO(), ns)
	return ns, err
}

func createByoHost(byoHostName string, byoHostNamespace string) (*infrastructurev1alpha4.ByoHost, error) {
	byoHost := &infrastructurev1alpha4.ByoHost{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ByoHost",
			APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      byoHostName,
			Namespace: byoHostNamespace,
		},
		Spec: infrastructurev1alpha4.ByoHostSpec{},
	}

	err := k8sClient.Create(context.TODO(), byoHost)
	return byoHost, err
}

func createSecret(bootstrapSecretName, stringDataValue, namespace string) (*corev1.Secret, error) {
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      bootstrapSecretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"value": []byte(stringDataValue),
		},
		Type: "cluster.x-k8s.io/secret",
	}

	err := k8sClient.Create(context.TODO(), secret)
	return secret, err
}

func createMachine(bootstrapSecret *string, machineName, namespace string) (*clusterv1.Machine, error) {
	machine := &clusterv1.Machine{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Machine",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      machineName,
			Namespace: namespace,
		},
		Spec: clusterv1.MachineSpec{
			Bootstrap: clusterv1.Bootstrap{
				DataSecretName: bootstrapSecret,
			},
			ClusterName: "default-test-cluster",
		},
	}

	err := k8sClient.Create(context.TODO(), machine)
	return machine, err
}

func createByoMachine(byoMachineName, namespace string, machine *clusterv1.Machine) (*infrastructurev1alpha4.ByoMachine, error) {
	byoMachine := &infrastructurev1alpha4.ByoMachine{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ByoMachine",
			APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      byoMachineName,
			Namespace: namespace,

			OwnerReferences: []metav1.OwnerReference{
				{
					Kind:       "Machine",
					Name:       machine.Name,
					APIVersion: "v1",
					UID:        machine.UID,
				},
			},
		},
		Spec: infrastructurev1alpha4.ByoMachineSpec{},
	}
	err := k8sClient.Create(context.TODO(), byoMachine)
	return byoMachine, err
}
