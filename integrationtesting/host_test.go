// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

//go:build integration

package integrationtesting

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/aibor/virtrun/internal/initprog"
	"github.com/aibor/virtrun/internal/initramfs"
	"github.com/aibor/virtrun/internal/qemu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostWithLibsNonZeroRC(t *testing.T) {
	t.Setenv("LD_LIBRARY_PATH", "../internal/initramfs/testdata/lib")

	binary, err := filepath.Abs("../internal/initramfs/testdata/bin/main")
	require.NoError(t, err)

	cmd, err := qemu.NewCommand(KernelArch)
	require.NoError(t, err)

	cmd.Kernel = KernelPath
	cmd.Verbose = Verbose

	init, err := initprog.For(KernelArch)
	require.NoError(t, err)

	irfs := initramfs.New(initramfs.WithVirtualInitFile(init))
	err = irfs.AddFile("/", "main", binary)
	require.NoError(t, err)

	err = irfs.AddRequiredSharedObjects("")
	require.NoError(t, err)

	cmd.Initramfs, err = irfs.WriteToTempFile(t.TempDir())
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	rc, err := cmd.Run(ctx, os.Stdout, os.Stderr)
	require.NoError(t, err)

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
			binary, err := filepath.Abs("testdata/bin/" + tt.bin)
			require.NoError(t, err)
			if KernelArch != runtime.GOARCH {
				t.Skipf("non matching architecture")
			}
			cmd, err := qemu.NewCommand(KernelArch)
			require.NoError(t, err)

			cmd.Kernel = KernelPath
			cmd.Verbose = Verbose
			cmd.Memory = 128
			cmd.InitArgs = tt.args

			init, err := initprog.For(KernelArch)
			require.NoError(t, err)

			irfs := initramfs.New(initramfs.WithVirtualInitFile(init))
			err = irfs.AddFile("/", "main", binary)
			require.NoError(t, err)

			cmd.Initramfs, err = irfs.WriteToTempFile(t.TempDir())
			require.NoError(t, err)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			t.Cleanup(cancel)

			rc, err := cmd.Run(ctx, os.Stdout, os.Stderr)

			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, 0, rc)
		})
	}
}
