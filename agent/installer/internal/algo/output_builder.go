// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package algo

type OutputBuilder interface {
	Out(string)
	Err(string)
	Cmd(string)
	Desc(string)
	Msg(string)
}
