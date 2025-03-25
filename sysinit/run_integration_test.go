// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration_sysinit

package sysinit_test

import (
	"os"
	"testing"

	"github.com/aibor/virtrun/sysinit"
	"golang.org/x/sys/unix"
)

func WithMountPoint(tb testing.TB, path string, fsType sysinit.FSType) {
	tb.Helper()

	err := sysinit.Mount(path, sysinit.MountOptions{FSType: fsType})
	if err != nil {
		tb.Fatalf("Failed to mount file system: %v", err)
	}

	tb.Cleanup(func() {
		if err := unix.Unmount(path, 0); err != nil {
			tb.Fatalf("Failed to unmount %s: %v", path, err)
		}
	})
}

func TestMain(m *testing.M) {
	sysinit.Run(
		sysinit.ExitCodeID.PrintFrom,
		func() error {
			if exitCode := m.Run(); exitCode != 0 {
				return sysinit.ExitError(exitCode)
			}

			return nil
		},
	)

	os.Exit(-1)
}
