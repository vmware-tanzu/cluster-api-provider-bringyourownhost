// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package algo

// OutputBuilder is an interface for building output as the algorithm runs.
type OutputBuilder interface {
	Out(string)
	Err(string)
	Cmd(string)
	Desc(string)
	Msg(string)
}
