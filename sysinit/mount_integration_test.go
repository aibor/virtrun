// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration_sysinit

package sysinit_test

import (
	"bufio"
	"os"
	"strings"
	"testing"

	"github.com/aibor/virtrun/sysinit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestMount(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		opts        sysinit.MountOptions
		expectedErr error
	}{
		{
			name:        "empty path",
			expectedErr: os.ErrNotExist,
		},
		{
			name:        "missing source",
			path:        "/test/some/path",
			expectedErr: unix.ENODEV,
		},
		{
			name: "nonexisting source",
			path: "/test/some/path",
			opts: sysinit.MountOptions{
				Source: "/test/non/existing",
			},
			expectedErr: unix.ENODEV,
		},
		{
			name: "nonexisting path",
			path: "/test/some/new/path",
			opts: sysinit.MountOptions{
				FSType: "tmpfs",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				err := unix.Unmount(tt.path, 0)
				if err != nil && tt.expectedErr == nil {
					t.Logf("Failed to unmount %s: %v", tt.path, err)
				}
			})

			err := sysinit.Mount(tt.path, tt.opts)
			require.ErrorIs(t, err, tt.expectedErr)

			if tt.expectedErr != nil {
				return
			}

			mountsFile, err := os.ReadFile("/proc/mounts")
			require.NoError(t, err)

			actual := map[string]string{}

			scanner := bufio.NewScanner(strings.NewReader(string(mountsFile)))
			for scanner.Scan() {
				columns := strings.Fields(scanner.Text())
				actual[columns[1]] = columns[2]
			}

			require.NoError(t, scanner.Err(), "must read mounts file")

			if assert.Contains(t, actual, tt.path) {
				assert.Equal(t, string(tt.opts.FSType), actual[tt.path])
			}
		})
	}
}

func TestMountAll(t *testing.T) {
	tests := []struct {
		name        string
		mounts      sysinit.MountPoints
		expectedErr error
	}{
		{
			name: "empty set",
		},
		{
			name: "invalid mount points",
			mounts: sysinit.MountPoints{
				"/test/somewhere": {},
			},
			expectedErr: unix.ENODEV,
		},
		{
			name: "invalid mount points may fail",
			mounts: sysinit.MountPoints{
				"/test/somewhereelse":  {},
				"/test/somewhereelse2": {},
			},
			expectedErr: sysinit.OptionalMountError{},
		},
		{
			name: "already mounted fails",
			mounts: sysinit.MountPoints{
				"/sys": {FSType: sysinit.FSTypeSys},
				"/run": {FSType: "tmpfs"},
			},
			expectedErr: unix.EBUSY,
		},
		{
			name: "valid mounts",
			mounts: sysinit.MountPoints{
				"/run": {FSType: "tmpfs"},
				"/tmp": {FSType: "tmpfs"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				for path := range tt.mounts {
					err := unix.Unmount(path, 0)
					if err != nil && tt.expectedErr == nil {
						t.Logf("Failed to unmount %s: %v", path, err)
					}
				}
			})

			err := sysinit.MountAll(tt.mounts)
			require.ErrorIs(t, err, tt.expectedErr)
		})
	}
}
