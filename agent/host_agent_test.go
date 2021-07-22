package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strconv"
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

	Context("When the host is unable to register with the API server", func() {
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
			Eventually(session).Should(gexec.Exit(0))
		})

		It("should return an error when invalid kubeconfig is passed in", func() {
			command := exec.Command(pathToHostAgentBinary, "--kubeconfig", fakedKubeConfig)
			session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ToNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
		})
	})

	Context("When the host agent is able to connect to API Server", func() {

		var (
			ns       *corev1.Namespace
			session  *gexec.Session
			err      error
			workDir  string
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

			workDir, err = ioutil.TempDir("", "host-agent-ut")
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			err = k8sClient.Delete(context.TODO(), ns)
			Expect(err).ToNot(HaveOccurred())
			os.RemoveAll(workDir)
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

			bootstrapSecretName := "bootstrap-secret-1"
			machineName := "test-machine-1"
			byoMachineName := "test-byomachine-1"

			fileName1 := path.Join(workDir, "file-1.txt")
			fileOriginContent1 := "some-content-1"
			fileNewContent1 := " run cmd"

			fileName2 := path.Join(workDir, "file-2.txt")
			fileOriginContent2 := "some-content-2"
			fileAppendContent2 := "some-content-append-2"
			filePermission2 := 0777
			isAppend2 := true

			fileName3 := path.Join(workDir, "file-3.txt")
			fileContent3 := "some-content-3"
			fileBase64Content3 := base64.StdEncoding.EncodeToString([]byte(fileContent3))

			fileName4 := path.Join(workDir, "file-4.txt")
			fileContent4 := "some-content-4"
			fileGzipContent4, err := gZipData([]byte(fileContent4))
			Expect(err).ToNot(HaveOccurred())
			fileGzipBase64Content4 := base64.StdEncoding.EncodeToString(fileGzipContent4)

			//Init second file
			err = ioutil.WriteFile(fileName2, []byte(fileOriginContent2), 0644)
			Expect(err).NotTo(HaveOccurred())

			byoHost := &infrastructurev1alpha4.ByoHost{}
			byoHostLookupKey := types.NamespacedName{Name: hostName, Namespace: ns.Name}
			Eventually(func() bool {
				err = k8sClient.Get(context.TODO(), byoHostLookupKey, byoHost)
				return err == nil
			}).ShouldNot(BeFalse())

			bootstrapSecretUnencoded := fmt.Sprintf(`write_files:
- path: %s
  content: %s
- path: %s
  permissions: '%s'
  content: %s
  append: %v
- path: %s
  content: %s
  encoding: base64
- path: %s
  encoding: gzip+base64
  content: %s
runCmd:
- echo -n '%s' >> %s`, fileName1, fileOriginContent1, fileName2, strconv.FormatInt(int64(filePermission2), 8), fileAppendContent2, isAppend2, fileName3, fileBase64Content3, fileName4, fileGzipBase64Content4, fileNewContent1, fileName1)

			secret, err := createSecret(bootstrapSecretName, bootstrapSecretUnencoded, ns.Name)
			Expect(err).ToNot(HaveOccurred())

			machine, err := createMachine(&secret.Name, machineName, ns.Name)
			Expect(err).ToNot(HaveOccurred())

			byoMachine, err := createByoMachine(byoMachineName, ns.Name, machine)
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

			//check first file's content
			Eventually(func() string {
				buffer, err := ioutil.ReadFile(fileName1)
				if err != nil {
					return ""
				}
				return string(buffer)
			}).Should(Equal(fileOriginContent1 + fileNewContent1))

			//check second file's content
			Eventually(func() string {
				buffer, err := ioutil.ReadFile(fileName2)
				if err != nil {
					return ""
				}
				return string(buffer)
			}).Should(Equal(fileOriginContent2 + fileAppendContent2))

			//check second file permission
			Eventually(func() bool {
				stats, err := os.Stat(fileName2)
				if err == nil && stats.Mode() == fs.FileMode(filePermission2) {
					return true
				}
				return false
			}).Should(BeTrue())

			//check if third files's content decoded in base64 way successfully
			Eventually(func() string {
				buffer, err := ioutil.ReadFile(fileName3)
				if err != nil {
					return ""
				}
				return string(buffer)
			}).Should(Equal(fileContent3))

			//check if fourth files's content decoded in gzip+base64 way successfully
			Eventually(func() string {
				buffer, err := ioutil.ReadFile(fileName4)
				if err != nil {
					return ""
				}
				return string(buffer)
			}).Should(Equal(fileContent4))

		})
	})

})

func gZipData(data []byte) ([]byte, error) {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)

	if _, err := gz.Write(data); err != nil {
		return nil, err
	}

	if err := gz.Flush(); err != nil {
		return nil, err
	}

	if err := gz.Close(); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

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
