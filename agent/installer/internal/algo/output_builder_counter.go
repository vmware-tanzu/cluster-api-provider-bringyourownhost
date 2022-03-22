// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package algo

// OutputBuilderCounter used to count the logs called by the OutputBuilder under various heads/types of output
type OutputBuilderCounter struct {
	LogCalledCnt int
}

// Out increments the log count for info/content output
func (c *OutputBuilderCounter) Out(str string) {
	c.LogCalledCnt++
}

// Err increments the log count for error output
func (c *OutputBuilderCounter) Err(str string) {
	c.LogCalledCnt++
}

// Cmd increments the log count for command output
func (c *OutputBuilderCounter) Cmd(str string) {
	c.LogCalledCnt++
}

// Desc increments the log count for description output
func (c *OutputBuilderCounter) Desc(str string) {
	c.LogCalledCnt++
}

// Msg increments the log count for message output
func (c *OutputBuilderCounter) Msg(str string) {
	c.LogCalledCnt++
}
