// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

//go:build integration

package integrationtesting_test

import (
	"bytes"
	"context"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/aibor/virtrun/internal/cmd"
	"github.com/aibor/virtrun/internal/qemu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostWithLibsNonZeroRC(t *testing.T) {
	t.Setenv("LD_LIBRARY_PATH", "../internal/initramfs/testdata/lib")

	binary, err := cmd.AbsoluteFilePath("../internal/initramfs/testdata/bin/main")
	require.NoError(t, err)

	args, err := cmd.NewArgs(KernelArch)
	require.NoError(t, err)

	args.Kernel = KernelPath
	args.Verbose = Verbose
	args.Binary = binary

	irfs, err := cmd.NewInitramfsArchive(args.InitramfsArgs)
	require.NoError(t, err)
	t.Cleanup(func() { _ = irfs.Cleanup() })

	cmd, err := cmd.NewQemuCommand(args.QemuArgs, irfs.Path)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	var stdOut, stdErr bytes.Buffer
	rc, err := cmd.Run(ctx, &stdOut, &stdErr)
	require.NoError(t, err)

	t.Log(stdOut.String())
	t.Log(stdErr.String())

	expectedRC := 73
	if KernelArch != runtime.GOARCH {
		expectedRC = 126
	}

	assert.Equal(t, expectedRC, rc)
}

func TestHostRCParsing(t *testing.T) {
	tests := []struct {
		name string
		bin  string
		args []string
		err  error
	}{
		{
			name: "return 0",
			bin:  "return",
			args: []string{"0"},
		},
		{
			name: "panic",
			bin:  "panic",
			err:  qemu.ErrGuestPanic,
		},
		{
			name: "oom",
			bin:  "oom",
			args: []string{"128"},
			err:  qemu.ErrGuestOom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			binary, err := cmd.AbsoluteFilePath("testdata/bin/" + tt.bin)
			require.NoError(t, err)

			if KernelArch != runtime.GOARCH {
				t.Skipf("non matching architecture")
			}

			args, err := cmd.NewArgs(KernelArch)
			require.NoError(t, err)

			args.Kernel = KernelPath
			args.Verbose = Verbose
			args.Binary = binary

			args.Memory.Value = 128
			args.InitArgs = tt.args

			irfs, err := cmd.NewInitramfsArchive(args.InitramfsArgs)
			require.NoError(t, err)
			t.Cleanup(func() { _ = irfs.Cleanup() })

			cmd, err := cmd.NewQemuCommand(args.QemuArgs, irfs.Path)
			require.NoError(t, err)

			t.Log(cmd.Args())

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			t.Cleanup(cancel)

			var stdOut, stdErr bytes.Buffer
			rc, err := cmd.Run(ctx, os.Stdout, os.Stderr)

			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)

				return
			}

			require.NoError(t, err)

			t.Log(stdOut.String())
			t.Log(stdErr.String())

			assert.Equal(t, 0, rc)
		})
	}
}
