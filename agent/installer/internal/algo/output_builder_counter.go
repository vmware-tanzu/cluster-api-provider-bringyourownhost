// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package algo

type OutputBuilderCounter struct {
	LogCalledCnt int
}

func (c *OutputBuilderCounter) Out(str string) {
	c.LogCalledCnt++
}

func (c *OutputBuilderCounter) Err(str string) {
	c.LogCalledCnt++
}

func (c *OutputBuilderCounter) Cmd(str string) {
	c.LogCalledCnt++
}

func (c *OutputBuilderCounter) Desc(str string) {
	c.LogCalledCnt++
}

func (c *OutputBuilderCounter) Msg(str string) {
	c.LogCalledCnt++
}
