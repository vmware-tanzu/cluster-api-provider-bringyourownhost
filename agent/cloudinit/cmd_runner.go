// Copyright 2021 VMware, Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package cloudinit

import (
	"fmt"
	"os/exec"
	"strings"
)

//counterfeiter:generate . ICmdRunner
type ICmdRunner interface {
	RunCmd(string) error
}

type CmdRunner struct {
}

func (r CmdRunner) RunCmd(cmd string) error {
	subStrs := []string{"kubeadm init", "kubeadm join"}

	for _, subStr := range subStrs {
		if strings.Contains(cmd, subStr) {
			index := strings.Index(cmd, subStr)
			index += len(subStr)
			newCmd := cmd[:index] + " --ignore-preflight-errors=all " + cmd[index:]
			cmd = newCmd
		}
	}
	command := exec.Command("/bin/sh", "-c", cmd)
	output, err := command.Output()
	fmt.Println(string(output))
	return err
}
