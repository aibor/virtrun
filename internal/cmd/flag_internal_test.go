// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"io"
	"testing"

	"github.com/aibor/virtrun/internal/qemu"
	"github.com/aibor/virtrun/internal/sys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlags_ParseArgs(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectedFlags *flags
		expecterErr   error
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
			expectedFlags: &flags{
				CPUType: "max",
				Memory:  256,
				NumCPU:  1,
				Version: true,
			},
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
			name: "empty additional file resets list",
			args: []string{
				"-kernel=/boot/this",
				"-addFile=/path",
				"-addFile=",
				"-addFile=/otherpath",
				"-addFile=/third/path",
				"bin.test",
			},
			expectedFlags: &flags{
				ExecutablePath: sys.MustAbsolutePath("bin.test"),
				DataFilePaths: []string{
					"/otherpath",
					"/third/path",
				},
				KernelPath: "/boot/this",
				CPUType:    "max",
				Memory:     256,
				NumCPU:     1,
				InitArgs:   []string{},
			},
		},
		{
			name: "debug",
			args: []string{
				"-kernel=/boot/this",
				"-debug",
				"bin.test",
			},
			expectedFlags: &flags{
				ExecutablePath: sys.MustAbsolutePath("bin.test"),
				KernelPath:     "/boot/this",
				CPUType:        "max",
				Memory:         256,
				NumCPU:         1,
				InitArgs:       []string{},
				Debug:          true,
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
				"-test.coverprofile=/some/file/cover.out",
			},
			expectedFlags: &flags{
				ExecutablePath: sys.MustAbsolutePath("bin.test"),
				DataFilePaths: []string{
					"/file2",
					"/dir/file3",
				},
				NoGoTestFlags: true,
				Standalone:    true,
				KeepInitramfs: true,
				KernelPath:    "/boot/this",
				CPUType:       "host",
				Machine:       "pc",
				TransportType: qemu.TransportTypeMMIO,
				Memory:        269,
				NoKVM:         true,
				NumCPU:        7,
				InitArgs: []string{
					"-test.paniconexit0",
					"-test.v=true",
					"-test.timeout=10m0s",
					"-test.coverprofile=/some/file/cover.out",
				},
				GuestVerbose: true,
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
			expectedFlags: &flags{
				ExecutablePath: sys.MustAbsolutePath("bin.test"),
				KernelPath:     "/boot/this",
				CPUType:        "max",
				Memory:         256,
				NumCPU:         1,
				InitArgs: []string{
					"-test.paniconexit0",
					"another.file",
					"-x",
					"-standalone",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags, err := parseArgs(tt.args, io.Discard)
			require.ErrorIs(t, err, tt.expecterErr)

			if tt.expecterErr != nil {
				return
			}

			assert.Equal(t, tt.expectedFlags, flags)
		})
	}
}
