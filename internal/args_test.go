// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package internal_test

import (
	"io"
	"testing"

	"github.com/aibor/virtrun/internal"
	"github.com/aibor/virtrun/internal/qemu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArgsParseArgs(t *testing.T) {
	absBinPath, err := internal.AbsoluteFilePath("bin.test")
	require.NoError(t, err)

	tests := []struct {
		name     string
		args     []string
		expected internal.Args
		errMsg   string
	}{
		{
			name: "help",
			args: []string{
				"-help",
			},
			errMsg: "flag: help requested",
		},
		{
			name: "version",
			args: []string{
				"-version",
			},
			errMsg: "flag: help requested",
		},
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
			expected: internal.Args{
				InitramfsArgs: internal.InitramfsArgs{
					Binary: absBinPath,
				},
				QemuArgs: internal.QemuArgs{
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
				"-transport", "mmio",
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
			expected: internal.Args{
				InitramfsArgs: internal.InitramfsArgs{
					Binary: absBinPath,
					Files: []string{
						"/file2",
						"/dir/file3",
					},
					Standalone:    true,
					KeepInitramfs: true,
				},
				QemuArgs: internal.QemuArgs{
					Kernel:        "/boot/this",
					CPU:           "host",
					Machine:       "pc",
					TransportType: qemu.TransportTypeMMIO,
					Memory:        internal.LimitedUintFlag{Value: 269},
					NoKVM:         true,
					SMP:           internal.LimitedUintFlag{Value: 7},
					InitArgs: []string{
						"-test.paniconexit0",
						"-test.v=true",
						"-test.timeout=10m0s",
					},
					Verbose:             true,
					NoGoTestFlagRewrite: true,
				},
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
			expected: internal.Args{
				InitramfsArgs: internal.InitramfsArgs{
					Binary: absBinPath,
				},
				QemuArgs: internal.QemuArgs{
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
		t.Run(tt.name, func(t *testing.T) {
			args := internal.Args{}

			err := args.ParseArgs("self", tt.args, io.Discard)

			if tt.errMsg != "" {
				assert.ErrorContains(t, err, tt.errMsg)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, args)
		})
	}
}
