// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package internal_test

import (
	"flag"
	"io"
	"testing"

	"github.com/aibor/virtrun/internal"
	"github.com/aibor/virtrun/internal/qemu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigParseArgs(t *testing.T) {
	absBinPath, err := internal.AbsoluteFilePath("bin.test")
	require.NoError(t, err)

	tests := []struct {
		name        string
		args        []string
		expected    internal.Config
		expecterErr error
	}{
		{
			name: "help",
			args: []string{
				"-help",
			},
			expecterErr: flag.ErrHelp,
		},
		{
			name: "version",
			args: []string{
				"-version",
			},
			expecterErr: flag.ErrHelp,
		},
		{
			name: "no kernel",
			args: []string{
				"bin.test",
			},
			expecterErr: &internal.ParseArgsError{},
		},
		{
			name: "no binary",
			args: []string{
				"-kernel=/boot/this",
			},
			expecterErr: &internal.ParseArgsError{},
		},
		{
			name: "additional file is empty",
			args: []string{
				"-kernel=/boot/this",
				"-addFile=",
				"bin.test",
			},
			expecterErr: &internal.ParseArgsError{},
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
			expected: internal.Config{
				Initramfs: internal.InitramfsConfig{
					Binary: absBinPath,
				},
				Qemu: internal.QemuConfig{
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
			expected: internal.Config{
				Initramfs: internal.InitramfsConfig{
					Binary: absBinPath,
					Files: []string{
						"/file2",
						"/dir/file3",
					},
					StandaloneInit: true,
					Keep:           true,
				},
				Qemu: internal.QemuConfig{
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
			expected: internal.Config{
				Initramfs: internal.InitramfsConfig{
					Binary: absBinPath,
				},
				Qemu: internal.QemuConfig{
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
			cfg := internal.Config{}

			err := cfg.ParseArgs("self", tt.args, io.Discard)
			require.ErrorIs(t, err, tt.expecterErr)

			if tt.expecterErr != nil {
				return
			}

			assert.Equal(t, tt.expected, cfg)
		})
	}
}
