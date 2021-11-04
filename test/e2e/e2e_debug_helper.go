// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/docker/docker/api/types"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DefaultFileMode                       fs.FileMode = 0777
	ReadByohControllerManagerLogShellFile string      = "/tmp/read-byoh-controller-manager-log.sh"
	ReadAllPodsShellFile                  string      = "/tmp/read-all-pods.sh"
)

type CaseContext struct {
	ctx              context.Context
	clusterProxy     framework.ClusterProxy
	cancelWatches    context.CancelFunc
	CaseName         string
	ClusterConName   string
	ClusterName      string
	SpecName         string
	Namespace        *corev1.Namespace
	ClusterResources *clusterctl.ApplyClusterTemplateAndWaitResult
}

type CollectInfoContext struct {
	AgentLogList     []string
	DeploymentLogDir string
}

type WriteDeploymentLogContext struct {
	DeploymentName      string
	DeploymentNamespace string
	ContainerName       string
}

func WriteDockerLog(output types.HijackedResponse, outputFile string) *os.File {
	s := make(chan string)
	e := make(chan error)
	buf := bufio.NewReader(output.Reader)
	f, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY, DefaultFileMode)
	if err != nil {
		Showf("OpenFile %s failed, Get err %v", outputFile, err)
		return nil
	}

	go func() {
		for {
			var line []byte
			line, _, err = buf.ReadLine()
			if err != nil {
				// will be quit by this err: read unix @->/run/docker.sock: use of closed network connection
				e <- err
				break
			} else {
				s <- string(line)
			}
		}
	}()

	go func() {
		for {
			select {
			case line := <-s:
				// ignore to print the error: file already closed
				_, err = f.WriteString(line + "\n")
				if err == nil {
					_ = f.Sync()
				}
			case err := <-e:
				// Please ignore this error if you see it in output
				Showf("Get err %v", err)
				return
			}
		}
	}()

	return f
}

func Showf(format string, a ...interface{}) {
	fmt.Printf(format, a...)
	fmt.Printf("\n")
}

func ShowFileContent(fileName string) {
	content, err := ioutil.ReadFile(fileName)
	if err != nil {
		Showf("ioutil.ReadFile %s return failed: Get err %v", fileName, err)
		return
	}

	Showf("######################Start: Content of %s##################", fileName)
	Showf("%s", string(content))
	Showf("######################End: Content of %s##################", fileName)
}

func ExecuteShellScript(shellFileName string) {
	cmd := exec.Command("/bin/sh", "-x", shellFileName)
	output, err := cmd.Output()
	if err != nil {
		Showf("execute %s return failed: Get err %v, output: %s", shellFileName, err, output)
		return
	}
	Showf("#######################Start: execute result of %s##################", shellFileName)
	Showf("%s", string(output))
	Showf("######################End: execute result of %s##################", shellFileName)
}

func WriteShellScript(shellFileName string, shellFileContent []string) {
	f, err := os.OpenFile(shellFileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, DefaultFileMode)
	if err != nil {
		Showf("Open %s return failed: Get err %v", shellFileName, err)
		return
	}

	defer f.Close()

	for _, line := range shellFileContent {
		if _, err = f.WriteString(line); err != nil {
			Showf("Write content %s return failed: Get err %v", line, err)
			return
		}
		if _, err = f.WriteString("\n"); err != nil {
			Showf("Write LF return failed: Get err %v", err)
			return
		}
	}
}

func WriteDeploymentLogs(caseContextData *CaseContext, collectInfoData *CollectInfoContext, writeDeploymentLogData *WriteDeploymentLogContext) {
	ctx := caseContextData.ctx
	clusterProxy := caseContextData.clusterProxy
	deploymentLogDir := collectInfoData.DeploymentLogDir
	deploymentNamespace := writeDeploymentLogData.DeploymentNamespace
	deploymentName := writeDeploymentLogData.DeploymentName
	containerName := writeDeploymentLogData.ContainerName

	deployment := &appsv1.Deployment{}
	key := client.ObjectKey{
		Namespace: deploymentNamespace,
		Name:      deploymentName,
	}

	if err := clusterProxy.GetClient().Get(ctx, key, deployment); err != nil {
		Showf("failed to get deployment %s/%s: %v", deploymentNamespace, deploymentName, err)
		return
	}

	selector, err := metav1.LabelSelectorAsMap(deployment.Spec.Selector)
	if err != nil {
		Showf("failed to get selector: %v", err)
		return
	}

	podList := &corev1.PodList{}
	if err = clusterProxy.GetClient().List(ctx, podList, client.InNamespace(deploymentNamespace), client.MatchingLabels(selector)); err != nil {
		Showf("failed to List pods in namespace %s : %v", deploymentNamespace, err)
		return
	}

	pods := podList.Items
	containers := deployment.Spec.Template.Spec.Containers

	os.RemoveAll(deploymentLogDir)
	if err = os.MkdirAll(deploymentLogDir, DefaultFileMode); err != nil {
		Showf("failed to create dir %s : %v", deploymentLogDir, err)
		return
	}

	for i := range pods {
		for j := range containers {
			if containers[j].Name != containerName {
				continue
			}
			go func(pod corev1.Pod, container corev1.Container) {
				logFile := path.Join(deploymentLogDir, pod.Name+"-"+container.Name+".log")
				var f *os.File
				f, err = os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY, DefaultFileMode)
				if err != nil {
					Showf("failed to open %s : %v", logFile, err)
					return
				}
				defer f.Close()

				opts := &corev1.PodLogOptions{
					Container: container.Name,
					Follow:    true,
				}
				var podLogs io.ReadCloser
				podLogs, err = clusterProxy.GetClientSet().CoreV1().Pods(deploymentNamespace).GetLogs(pod.Name, opts).Stream(ctx)
				if err != nil {
					Showf("failed to get the log of pod %s: %v", pod.Name, err)
					return
				}
				defer podLogs.Close()

				out := bufio.NewWriter(f)
				defer out.Flush()

				_, err = out.ReadFrom(podLogs)
				if err != nil && err != io.ErrUnexpectedEOF {
					Showf("Got error while streaming logs for pod %s/%s, container %s: %v", deploymentNamespace, pod.Name, container.Name, err)
					return
				}
			}(pods[i], containers[j])
		}
	}
}

func ShowDeploymentLogs(logDir string) {
	logFiles, err := filepath.Glob(logDir + "/*")
	if err != nil {
		Showf("failed to list all file from dir %s: %v", logDir, err)
		return
	}
	for _, logFile := range logFiles {
		ShowFileContent(logFile)
	}
}

func ShowInfoBeforeCaseQuit() {
	// show swap status
	// showFileContent("/proc/swaps")

	// show the status of  all pods
	shellContent := []string{
		"kubectl get pods --all-namespaces --kubeconfig /tmp/mgmt.conf",
	}
	WriteShellScript(ReadAllPodsShellFile, shellContent)
	ShowFileContent(ReadAllPodsShellFile)
	ExecuteShellScript(ReadAllPodsShellFile)
}

func CollectInfo(caseContextData *CaseContext, collectInfoData *CollectInfoContext) {
	// collecting deployment logs in go rountinue
	WriteDeploymentLogs(caseContextData, collectInfoData, &WriteDeploymentLogContext{
		DeploymentName:      "byoh-controller-manager",
		DeploymentNamespace: "byoh-system",
		ContainerName:       "manager",
	})
}

// The jobs "write agent log" and "write Deployment logs" are running in go rountine.
// Move jobs "show agent log" and "show Deployment logs" here is to get the full logs.
// Because only after agent and deployment quit, then it can get full logs.
func ShowInfoAfterCaseQuit(collectInfoData *CollectInfoContext) {
	// show the agent log
	for _, agentLogFile := range collectInfoData.AgentLogList {
		ShowFileContent(agentLogFile)
	}

	// show the deployment's pods log
	ShowDeploymentLogs(collectInfoData.DeploymentLogDir)
}

func RemoveLogs(collectInfoData *CollectInfoContext) {
	for _, agentLogFile := range collectInfoData.AgentLogList {
		os.Remove(agentLogFile)
	}

	os.Remove(ReadByohControllerManagerLogShellFile)
	os.Remove(ReadAllPodsShellFile)
	os.RemoveAll(collectInfoData.DeploymentLogDir)
}
