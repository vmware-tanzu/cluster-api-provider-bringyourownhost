// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"os/exec"

	"github.com/docker/docker/api/types"
)

const (
	// DefaultFileMode the default file mode of files created for tests
	DefaultFileMode fs.FileMode = 0777
	// ReadByohControllerManagerLogShellFile location of script to read the controller manager log
	ReadByohControllerManagerLogShellFile string = "/tmp/read-byoh-controller-manager-log.sh"
	// ReadAllPodsShellFile location of script to read all pods logs
	ReadAllPodsShellFile string = "/tmp/read-all-pods.sh"
)

// WriteDockerLog redirects the docker logs to the given file
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
			line, _, err := buf.ReadLine()
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
				_, err2 := f.WriteString(line + "\n")
				if err2 != nil {
					Showf("Write String to file failed, err2=%v", err2)
				}
				_ = f.Sync()
			case err := <-e:
				// Please ignore this error if you see it in output
				Showf("Get err %v", err)
				return
			}
		}
	}()

	return f
}

// Showf prints formatted string to stdout
func Showf(format string, a ...interface{}) {
	fmt.Printf(format, a...)
	fmt.Printf("\n")
}

// ShowFileContent prints to stdout the content of the given file
func ShowFileContent(fileName string) {
	content, err := os.ReadFile(fileName)
	if err != nil {
		Showf("ioutil.ReadFile %s return failed: Get err %v", fileName, err)
		return
	}

	Showf("######################Start: Content of %s##################", fileName)
	Showf("%s", string(content))
	Showf("######################End: Content of %s##################", fileName)
}

// ExecuteShellScript executes a given shell script file location
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

// WriteShellScript writes shell script contents/commands to the given file location
func WriteShellScript(shellFileName string, shellFileContent []string) {
	f, err := os.OpenFile(shellFileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, DefaultFileMode)
	if err != nil {
		Showf("Open %s return failed: Get err %v", shellFileName, err)
		return
	}

	defer func() {
		deferredErr := f.Close()
		if deferredErr != nil {
			Showf("Close %s return failed: Get err %v", shellFileName, deferredErr)
		}
	}()

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

// ShowInfo shows all the pods status, agent logs, and controller manager logs
func ShowInfo(allAgentLogFiles []string) {
	// show swap status
	// showFileContent("/proc/swaps")

	// show the status of  all pods
	shellContent := []string{
		"kubectl get pods --all-namespaces --kubeconfig /tmp/mgmt.conf",
	}
	WriteShellScript(ReadAllPodsShellFile, shellContent)
	ShowFileContent(ReadAllPodsShellFile)
	ExecuteShellScript(ReadAllPodsShellFile)

	// show the agent log
	for _, agentLogFile := range allAgentLogFiles {
		ShowFileContent(agentLogFile)
	}

	// show byoh-controller-manager logs
	shellContent = []string{
		"podNamespace=`kubectl get pods --all-namespaces --kubeconfig /tmp/mgmt.conf | grep byoh-controller-manager | awk '{print $1}'`",
		"podName=`kubectl get pods --all-namespaces --kubeconfig /tmp/mgmt.conf | grep byoh-controller-manager | awk '{print $2}'`",
		"kubectl logs -n ${podNamespace} ${podName} --kubeconfig /tmp/mgmt.conf -c manager",
	}

	WriteShellScript(ReadByohControllerManagerLogShellFile, shellContent)
	ShowFileContent(ReadByohControllerManagerLogShellFile)
	ExecuteShellScript(ReadByohControllerManagerLogShellFile)
}
