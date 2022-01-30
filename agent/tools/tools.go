// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build tools
// +build tools

package tools

import (
	_ "github.com/maxbrunsfeld/counterfeiter/v6"
)

// This file imports packages that are used when running go generate, or used
// during the development process but not otherwise depended on by built code.
