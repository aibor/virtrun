// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aibor/virtrun/internal/qemu"
)

func TestParseArgs(t *testing.T) {
	absBinPath, err := filepath.Abs("bin.test")
	require.NoError(t, err)

	tests := []struct {
		name     string
		args     []string
		expected config
		errMsg   string
	}{
		{
			name: "no kernel",
			args: []string{
				"bin.test",
			},
			errMsg: "no kernel given",
		},
		{
			name: "no binary",
			args: []string{
				"-kernel=/boot/this",
			},
			errMsg: "no binary given",
		},
		{
			name: "additional file is empty",
			args: []string{
				"-kernel=/boot/this",
				"-addFile=",
				"bin.test",
			},
			errMsg: "file path must not be empty",
		},
		{
			name: "simple go test invocation",
			args: []string{
				"-kernel=/boot/this",
				"bin.test",
				"-test.paniconexit0",
				"-test.v=true",
				"-test.timeout=10m0s",
			},
			expected: config{
				binary: absBinPath,
				cmd: &qemu.Command{
					Kernel: "/boot/this",
					InitArgs: []string{
						"-test.paniconexit0",
						"-test.v=true",
						"-test.timeout=10m0s",
					},
				},
			},
		},
		{
			name: "go test invocation with virtrun flags",
			args: []string{
				"-kernel=/boot/this",
				"-cpu", "host",
				"-machine=pc",
				"-transport", "2",
				"-memory=269",
				"-verbose",
				"-smp", "7",
				"-nokvm=true",
				"-standalone",
				"-noGoTestFlagRewrite",
				"-keepInitramfs",
				"-addFile", "/file2",
				"-addFile", "/dir/file3",
				"bin.test",
				"-test.paniconexit0",
				"-test.v=true",
				"-test.timeout=10m0s",
			},
			expected: config{
				binary: absBinPath,
				files: []string{
					"/file2",
					"/dir/file3",
				},
				cmd: &qemu.Command{
					Kernel:        "/boot/this",
					CPU:           "host",
					Machine:       "pc",
					TransportType: qemu.TransportTypeMMIO,
					Memory:        269,
					Verbose:       true,
					NoKVM:         true,
					SMP:           7,
					InitArgs: []string{
						"-test.paniconexit0",
						"-test.v=true",
						"-test.timeout=10m0s",
					},
				},
				standalone:          true,
				noGoTestFlagRewrite: true,
				keepInitramfs:       true,
			},
		},
		{
			name: "flag parsing stops at flags after binary file",
			args: []string{
				"-kernel=/boot/this",
				"bin.test",
				"-test.paniconexit0",
				"another.file",
				"-x",
				"-standalone",
			},
			expected: config{
				binary: absBinPath,
				cmd: &qemu.Command{
					Kernel: "/boot/this",
					InitArgs: []string{
						"-test.paniconexit0",
						"another.file",
						"-x",
						"-standalone",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			cfg := config{
				cmd: &qemu.Command{},
			}

			execArgs := append([]string{"self"}, tt.args...)
			err := cfg.parseArgs(execArgs)

			if tt.errMsg != "" {
				assert.ErrorContains(t, err, tt.errMsg)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, cfg)
		})
	}
}
