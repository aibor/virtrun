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

	"github.com/aibor/virtrun/internal"
	"github.com/aibor/virtrun/internal/qemu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostWithLibsNonZeroRC(t *testing.T) {
	t.Setenv("LD_LIBRARY_PATH", "../internal/initramfs/testdata/lib")

	binary, err := internal.AbsoluteFilePath("../internal/initramfs/testdata/bin/main")
	require.NoError(t, err)

	args, err := internal.NewArgs(KernelArch)
	require.NoError(t, err)

	args.Kernel = KernelPath
	args.Verbose = Verbose
	args.Binary = binary

	irfs, err := internal.NewInitramfsArchive(args.InitramfsArgs)
	require.NoError(t, err)
	t.Cleanup(func() { _ = irfs.Cleanup() })

	cmd, err := internal.NewQemuCommand(args.QemuArgs, irfs.Path)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	var (
		stdOut, stdErr bytes.Buffer
		cmdErr         *qemu.CommandError
	)

	err = cmd.Run(ctx, &stdOut, &stdErr)

	t.Log(stdOut.String())
	t.Log(stdErr.String())

	require.ErrorAs(t, err, &cmdErr)

	expectedRC := 73
	if KernelArch != runtime.GOARCH {
		expectedRC = 126
	}

	assert.Equal(t, expectedRC, cmdErr.ExitCode)
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
			name: "return 1",
			bin:  "return",
			args: []string{"1"},
			err:  qemu.ErrGuestNonZeroExitCode,
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
			binary, err := internal.AbsoluteFilePath("testdata/bin/" + tt.bin)
			require.NoError(t, err)

			if KernelArch != runtime.GOARCH {
				t.Skipf("non matching architecture")
			}

			args, err := internal.NewArgs(KernelArch)
			require.NoError(t, err)

			args.Kernel = KernelPath
			args.Verbose = Verbose
			args.Binary = binary
			args.Memory.Value = 128
			args.InitArgs = tt.args

			irfs, err := internal.NewInitramfsArchive(args.InitramfsArgs)
			require.NoError(t, err)
			t.Cleanup(func() { _ = irfs.Cleanup() })

			cmd, err := internal.NewQemuCommand(args.QemuArgs, irfs.Path)
			require.NoError(t, err)

			t.Log(cmd.Args())

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			t.Cleanup(cancel)

			var stdOut, stdErr bytes.Buffer

			err = cmd.Run(ctx, os.Stdout, os.Stderr)

			t.Log(stdOut.String())
			t.Log(stdErr.String())

			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)

				return
			}

			require.NoError(t, err)
		})
	}
}
