// Copyright 2026 Platform9, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build !linux

package reconciler

// deleteIP is a no-op on non-Linux platforms — see host_reconciler_linux.go.
// Only here so this package builds and tests natively on a developer's
// non-Linux machine; the agent itself only ever runs on Linux BYOH hosts.
func deleteIP(ip, iface string) error {
	return nil
}
