// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"io"
	"testing"

	"github.com/aibor/virtrun/internal/qemu"
	"github.com/aibor/virtrun/internal/sys"
	"github.com/aibor/virtrun/internal/virtrun"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlags_ParseArgs(t *testing.T) {
	tests := []struct {
		name              string
		args              []string
		expectedSpec      *virtrun.Spec
		expectedDebugFlag bool
		expecterErr       error
	}{
		{
			name: "help",
			args: []string{
				"-help",
			},
			expecterErr: ErrHelp,
		},
		{
			name: "version",
			args: []string{
				"-version",
			},
			expecterErr: ErrHelp,
		},
		{
			name: "no kernel",
			args: []string{
				"bin.test",
			},
			expecterErr: &ParseArgsError{},
		},
		{
			name: "no binary",
			args: []string{
				"-kernel=/boot/this",
			},
			expecterErr: &ParseArgsError{},
		},
		{
			name: "additional file is empty",
			args: []string{
				"-kernel=/boot/this",
				"-addFile=",
				"bin.test",
			},
			expecterErr: &ParseArgsError{},
		},
		{
			name: "debug",
			args: []string{
				"-kernel=/boot/this",
				"-debug",
				"bin.test",
			},
			expectedSpec: &virtrun.Spec{
				Initramfs: virtrun.Initramfs{
					Binary: sys.MustAbsolutePath("bin.test"),
				},
				Qemu: virtrun.Qemu{
					Kernel:   "/boot/this",
					CPU:      "max",
					Memory:   256,
					SMP:      1,
					InitArgs: []string{},
				},
			},
			expectedDebugFlag: true,
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
			expectedSpec: &virtrun.Spec{
				Initramfs: virtrun.Initramfs{
					Binary: sys.MustAbsolutePath("bin.test"),
				},
				Qemu: virtrun.Qemu{
					Kernel: "/boot/this",
					CPU:    "max",
					Memory: 256,
					SMP:    1,
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
			expectedSpec: &virtrun.Spec{
				Initramfs: virtrun.Initramfs{
					Binary: sys.MustAbsolutePath("bin.test"),
					Files: []string{
						"/file2",
						"/dir/file3",
					},
					StandaloneInit: true,
					Keep:           true,
				},
				Qemu: virtrun.Qemu{
					Kernel:        "/boot/this",
					CPU:           "host",
					Machine:       "pc",
					TransportType: qemu.TransportTypeMMIO,
					Memory:        269,
					NoKVM:         true,
					SMP:           7,
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
			expectedSpec: &virtrun.Spec{
				Initramfs: virtrun.Initramfs{
					Binary: sys.MustAbsolutePath("bin.test"),
				},
				Qemu: virtrun.Qemu{
					Kernel: "/boot/this",
					CPU:    "max",
					Memory: 256,
					SMP:    1,
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
			flags := newFlags("test", io.Discard)

			err := flags.ParseArgs(tt.args)
			require.ErrorIs(t, err, tt.expecterErr)

			if tt.expecterErr != nil {
				return
			}

			assert.Equal(t, tt.expectedSpec, flags.spec, "spec")
			assert.Equal(t, tt.expectedDebugFlag, flags.Debug(), "debug flag")
		})
	}
}
