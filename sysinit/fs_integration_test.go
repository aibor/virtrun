// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package sysinit_test

import (
	"bufio"
	"os"
	"path/filepath"
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
				FSType: sysinit.FSTypeTmp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
				_ = unix.Unmount(tt.path, 0)
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
				"/test/somewhereelse": {
					MayFail: true,
				},
			},
		},
		{
			name: "valid mounts",
			mounts: sysinit.MountPoints{
				"/dev":                {FSType: sysinit.FSTypeDevTmp},
				"/dev/shm":            {FSType: sysinit.FSTypeTmp},
				"/proc":               {FSType: sysinit.FSTypeProc},
				"/run":                {FSType: sysinit.FSTypeTmp},
				"/sys":                {FSType: sysinit.FSTypeSys},
				"/sys/fs/bpf":         {FSType: sysinit.FSTypeDebug},
				"/sys/kernel/debug":   {FSType: sysinit.FSTypeDebug},
				"/sys/kernel/tracing": {FSType: sysinit.FSTypeTracing},
				"/tmp":                {FSType: sysinit.FSTypeTmp},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sysinit.MountAll(tt.mounts)
			require.ErrorIs(t, err, tt.expectedErr)

			if tt.expectedErr != nil {
				return
			}

			for path := range tt.mounts {
				_ = unix.Unmount(path, 0)
			}
		})
	}
}

func TestCreateSymlinks(t *testing.T) {
	tests := []struct {
		name        string
		symlinks    sysinit.Symlinks
		expectedErr error
	}{
		{
			name: "empty set",
		},
		{
			name: "invalid link",
			symlinks: sysinit.Symlinks{
				"": "/tmp",
			},
			expectedErr: os.ErrNotExist,
		},
		{
			name: "valid symlinks",
			symlinks: sysinit.Symlinks{
				"/dev/core":   "/proc/kcore",
				"/dev/fd":     "/proc/self/fd/",
				"/dev/rtc":    "rtc0",
				"/dev/stdin":  "/proc/self/fd/0",
				"/dev/stdout": "/proc/self/fd/1",
				"/dev/stderr": "/proc/self/fd/2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sysinit.CreateSymlinks(tt.symlinks)
			require.ErrorIs(t, err, tt.expectedErr)

			if tt.expectedErr != nil {
				return
			}

			for link, expectedTarget := range tt.symlinks {
				target, err := os.Readlink(link)
				if assert.NoError(t, err, "link should be readable") {
					assert.Equal(t, expectedTarget, target,
						"link target should be as expected")

					_ = os.Remove(link)
				}
			}
		})
	}
}

func TestListRegularFiles(t *testing.T) {
	tempDir := t.TempDir()

	err := os.MkdirAll(filepath.Join(tempDir, "one", "two"), os.ModePerm)
	require.NoError(t, err)

	_, err = os.Create(filepath.Join(tempDir, "one", "file1"))
	require.NoError(t, err)

	_, err = os.Create(filepath.Join(tempDir, "one", "two", "file2"))
	require.NoError(t, err)

	actual, err := sysinit.ListRegularFiles(tempDir)
	require.NoError(t, err)

	expected := []string{
		filepath.Join(tempDir, "one", "file1"),
		filepath.Join(tempDir, "one", "two", "file2"),
	}

	assert.Equal(t, expected, actual)
}
