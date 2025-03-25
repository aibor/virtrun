// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration_sysinit

package sysinit_test

import (
	"bufio"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aibor/virtrun/sysinit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

var testModules = flag.String("testModules", "", "module names to test")

func passedModules() []string {
	var passedModules []string
	if *testModules != "" {
		passedModules = strings.Split(*testModules, ",")
	}

	return passedModules
}

func TestLoadModule(t *testing.T) {
	WithMountPoint(t, "/proc", sysinit.FSTypeProc)
	WithMountPoint(t, "/tmp", sysinit.FSTypeTmp)

	tempDir := t.TempDir()
	fileTxtPath := filepath.Join(tempDir, "mod.txt")

	fileTxt, err := os.Create(fileTxtPath)
	require.NoError(t, err)

	_ = fileTxt.Close()

	modules, err := os.ReadDir("/lib/modules/")
	require.NoError(t, err)

	validMod := "/lib/modules/" + modules[0].Name()

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
		{
			name: "valid mod",
			path: validMod,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				for _, module := range passedModules() {
					_ = unix.DeleteModule(module, 0)
				}
			})

			err := sysinit.LoadModule(tt.path, tt.params)
			require.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

func TestLoadModules(t *testing.T) {
	WithMountPoint(t, "/proc", sysinit.FSTypeProc)
	WithMountPoint(t, "/tmp", sysinit.FSTypeTmp)

	tests := []struct {
		name            string
		dir             func(t *testing.T) string
		expectedModules []string
		expectedErr     error
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
				return t.TempDir() + "/*"
			},
		},
		{
			name: "dir with invalid ext file",
			dir: func(t *testing.T) string {
				t.Helper()
				fileName := t.TempDir() + "/mod.txt"
				err := os.WriteFile(fileName, []byte{}, 0o600)
				require.NoError(t, err)

				return fileName
			},
			expectedErr: unix.EINVAL,
		},
		{
			name: "dir with valid ext empty file",
			dir: func(t *testing.T) string {
				t.Helper()
				fileName := t.TempDir() + "/mod.ko"
				err := os.WriteFile(fileName, []byte{}, 0o600)
				require.NoError(t, err)

				return fileName
			},
			expectedErr: unix.EINVAL,
		},
		{
			name: "real dir",
			dir: func(t *testing.T) string {
				t.Helper()
				return "/lib/modules/*"
			},
			expectedModules: passedModules(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				for _, module := range passedModules() {
					_ = unix.DeleteModule(module, 0)
				}
			})

			dir := tt.dir(t)

			err := sysinit.LoadModules(dir)
			require.ErrorIs(t, err, tt.expectedErr)

			modules, err := os.ReadFile("/proc/modules")
			require.NoError(t, err, "must read modules")

			actual := []string{}

			scanner := bufio.NewScanner(strings.NewReader(string(modules)))
			for scanner.Scan() {
				fields := strings.Fields(scanner.Text())
				actual = append(actual, fields[0])
			}

			require.NoError(t, scanner.Err(), "must read modules file")

			assert.ElementsMatch(t, tt.expectedModules, actual)
		})
	}
}
