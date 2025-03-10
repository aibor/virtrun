// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package sysinit_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aibor/virtrun/sysinit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestLoadModule(t *testing.T) {
	tempDir := t.TempDir()
	fileTxtPath := filepath.Join(tempDir, "mod.txt")

	fileTxt, err := os.Create(fileTxtPath)
	require.NoError(t, err)

	_ = fileTxt.Close()

	tests := []struct {
		name        string
		path        string
		params      string
		expectedErr error
	}{
		{
			name:        "empty path",
			expectedErr: os.ErrNotExist,
		},
		{
			name:        "non-existing path",
			path:        "/where/ever",
			expectedErr: os.ErrNotExist,
		},
		{
			name:        "unsupported extensions",
			path:        fileTxtPath,
			expectedErr: unix.EINVAL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sysinit.LoadModule(tt.path, tt.params)
			require.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

func TestLoadModules(t *testing.T) {
	tests := []struct {
		name        string
		dir         func(t *testing.T) string
		expectedErr error
	}{
		{
			name: "empty dir path",
			dir: func(t *testing.T) string {
				t.Helper()
				return ""
			},
		},
		{
			name: "empty dir",
			dir: func(t *testing.T) string {
				t.Helper()
				return t.TempDir()
			},
		},
		{
			name: "dir with invalid ext file",
			dir: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				err := os.WriteFile(dir+"/mod.txt", []byte{}, 0o600)
				require.NoError(t, err)

				return dir
			},
			expectedErr: unix.EINVAL,
		},
		{
			name: "dir with valid ext empty file",
			dir: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				err := os.WriteFile(dir+"/mod.ko", []byte{}, 0o600)
				require.NoError(t, err)

				return dir
			},
			expectedErr: unix.EINVAL,
		},
		{
			name: "real dir",
			dir: func(t *testing.T) string {
				t.Helper()
				return "/lib/modules"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.dir(t)

			err := sysinit.LoadModules(dir)
			require.ErrorIs(t, err, tt.expectedErr)

			if tt.expectedErr != nil || dir == "" {
				return
			}

			entries, err := os.ReadDir(dir)
			require.NoError(t, err, "must read dir")

			modules, err := os.ReadFile("/proc/modules")
			require.NoError(t, err, "must read modules")

			t.Log(string(modules))

			assert.Equal(t, len(entries), strings.Count(string(modules), "\n"))
		})
	}
}
