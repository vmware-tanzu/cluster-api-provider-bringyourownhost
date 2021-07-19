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
	"os/user"
	"path"
	"strconv"
	"syscall"
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

			fileName := path.Join(workDir, "file.txt")
			fileOriginContent := "some-content-1"
			fileNewContent := " run cmd"

			byoHost := &infrastructurev1alpha4.ByoHost{}
			byoHostLookupKey := types.NamespacedName{Name: hostName, Namespace: ns.Name}
			Eventually(func() bool {
				err = k8sClient.Get(context.TODO(), byoHostLookupKey, byoHost)
				return err == nil
			}).ShouldNot(BeFalse())

			bootstrapSecretUnencoded := fmt.Sprintf(`write_files:
- path: %s
  content: %s
runCmd:
- echo -n '%s' >> %s`, fileName, fileOriginContent, fileNewContent, fileName)

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

			Eventually(func() string {
				buffer, err := ioutil.ReadFile(fileName)
				if err != nil {
					return ""
				}
				return string(buffer)
			}).Should(Equal(fileOriginContent + fileNewContent))

		})

		It("check if file created by bootstrap script in correct attributes", func() {
			bootstrapSecretName := "bootstrap-secret-2"
			machineName := "test-machine-2"
			byoMachineName := "test-byomachine-2"
			fileName := path.Join(workDir, "file-2.txt")
			fileOriginContent := "some-content-2"
			fileAppendContent := "some-content-append"
			filePermission := 0777
			userName := "root"
			groupName := "root"
			isAppend := true

			//Init file
			err = ioutil.WriteFile(fileName, []byte(fileOriginContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			bootstrapSecretUnencoded := fmt.Sprintf(`write_files:
- path: %s
  owner: %s:%s
  permissions: '%s'
  content: %s
  append: %v`, fileName, userName, groupName, strconv.FormatInt(int64(filePermission), 8), fileAppendContent, isAppend)

			byoHost := &infrastructurev1alpha4.ByoHost{}
			byoHostLookupKey := types.NamespacedName{Name: hostName, Namespace: ns.Name}
			Eventually(func() bool {
				err = k8sClient.Get(context.TODO(), byoHostLookupKey, byoHost)
				return err == nil
			}).ShouldNot(BeFalse())

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

			//check file content
			Eventually(func() string {
				buffer, err := ioutil.ReadFile(fileName)
				if err != nil {
					return ""
				}
				return string(buffer)
			}).Should(Equal(fileOriginContent + fileAppendContent))

			//check file permission
			Eventually(func() bool {
				stats, err := os.Stat(fileName)
				if err == nil && stats.Mode() == fs.FileMode(filePermission) {
					return true
				}
				return false
			}).Should(BeTrue())

			//check file owner
			Eventually(func() bool {
				stats, err := os.Stat(fileName)
				if err != nil {
					return false
				}
				stat := stats.Sys().(*syscall.Stat_t)

				userInfo, err := user.Lookup(userName)
				if err != nil {
					return false
				}

				uid, err := strconv.ParseUint(userInfo.Uid, 10, 32)
				if err != nil {
					return false
				}

				gid, err := strconv.ParseUint(userInfo.Gid, 10, 32)
				if err != nil {
					return false
				}

				if stat.Uid == uint32(uid) && stat.Gid == uint32(gid) {
					return true
				}
				return false

			}).Should(BeTrue())

		})

		It("check if file created by bootstrap script in correct encoding way", func() {
			bootstrapSecretName := "bootstrap-secret-3"
			machineName := "test-machine-3"
			byoMachineName := "test-byomachine-3"

			fileName1 := path.Join(workDir, "file-3-1.txt")
			fileContent1 := "some-content-3-1"
			fileBase64Content1 := base64.StdEncoding.EncodeToString([]byte(fileContent1))

			fileName2 := path.Join(workDir, "file-3-2.txt")
			fileContent2 := "some-content-3-2"
			fileGzipContent2, err := gZipData([]byte(fileContent2))
			Expect(err).ToNot(HaveOccurred())
			fileGzipBase64Content2 := base64.StdEncoding.EncodeToString(fileGzipContent2)

			bootstrapSecretUnencoded := fmt.Sprintf(`write_files:
- path: %s
  content: %s
  encoding: base64
- path: %s
  encoding: gzip+base64
  content: %s`, fileName1, fileBase64Content1, fileName2, fileGzipBase64Content2)

			byoHost := &infrastructurev1alpha4.ByoHost{}
			byoHostLookupKey := types.NamespacedName{Name: hostName, Namespace: ns.Name}
			Eventually(func() bool {
				err = k8sClient.Get(context.TODO(), byoHostLookupKey, byoHost)
				return err == nil
			}).ShouldNot(BeFalse())

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

			//bas64
			Eventually(func() string {
				buffer, err := ioutil.ReadFile(fileName1)
				if err != nil {
					return ""
				}
				return string(buffer)
			}).Should(Equal(fileContent1))

			//gzip+base64
			Eventually(func() string {
				buffer, err := ioutil.ReadFile(fileName2)
				if err != nil {
					return ""
				}
				return string(buffer)
			}).Should(Equal(fileContent2))

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
