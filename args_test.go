// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package main

import (
	"io"
	"testing"

	"github.com/aibor/virtrun/internal/qemu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArgsParseArgs(t *testing.T) {
	absBinPath, err := absoluteFilePath("bin.test")
	require.NoError(t, err)

	tests := []struct {
		name     string
		args     []string
		expected args
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
			expected: args{
				initramfsArgs: initramfsArgs{
					binary: absBinPath,
				},
				qemuArgs: qemuArgs{
					kernel: "/boot/this",
					initArgs: []string{
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
			expected: args{
				initramfsArgs: initramfsArgs{
					binary: absBinPath,
					files: []string{
						"/file2",
						"/dir/file3",
					},
					standalone:    true,
					keepInitramfs: true,
				},
				qemuArgs: qemuArgs{
					kernel:    "/boot/this",
					cpu:       "host",
					machine:   "pc",
					transport: transport{qemu.TransportTypeMMIO},
					memory:    limitedUintFlag{value: 269},
					noKVM:     true,
					smp:       limitedUintFlag{value: 7},
					initArgs: []string{
						"-test.paniconexit0",
						"-test.v=true",
						"-test.timeout=10m0s",
					},
					verbose:             true,
					noGoTestFlagRewrite: true,
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
			expected: args{
				initramfsArgs: initramfsArgs{
					binary: absBinPath,
				},
				qemuArgs: qemuArgs{
					kernel: "/boot/this",
					initArgs: []string{
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
			args := args{}

			err := args.parseArgs("self", tt.args, io.Discard)

			if tt.errMsg != "" {
				assert.ErrorContains(t, err, tt.errMsg)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, args)
		})
	}
}

func TestAddArgsFromEnv(t *testing.T) {
	tests := []struct {
		name   string
		env    string
		input  []string
		output []string
	}{
		{
			name:   "empty",
			env:    "",
			input:  []string{},
			output: []string{},
		},
		{
			name:   "only input, empty env",
			env:    "",
			input:  []string{"-kernel", "/boot/vmlinuz"},
			output: []string{"-kernel", "/boot/vmlinuz"},
		},
		{
			name:   "only env, empty input",
			env:    "-kernel /boot/vmlinuz",
			input:  []string{},
			output: []string{"-kernel", "/boot/vmlinuz"},
		},
		{
			name:   "both used",
			env:    "-kernel /boot/vmlinuz",
			input:  []string{"-verbose"},
			output: []string{"-kernel", "/boot/vmlinuz", "-verbose"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			varName := "VIRTRUN_ARGS"
			t.Setenv(varName, tt.env)
			assert.Equal(t, tt.output, prependEnvArgs(tt.input))
		})
	}
}
