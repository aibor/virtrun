// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

//go:build integration

//go:generate env CGO_ENABLED=0 go build -v -trimpath -buildvcs=false -o bin/ ./cmd/...

//go:generate -command guesttest env CGO_ENABLED=0 go test -c -cover -covermode atomic -coverpkg github.com/aibor/virtrun/sysinit ./guest/
//go:generate guesttest -c -tags integration_guest -o bin/guest.test
//go:generate guesttest -c -tags integration_guest,standalone -o bin/guest.standalone.test

package integration_test

import (
	"bytes"
	"context"
	"flag"
	"testing"
	"time"

	"github.com/aibor/virtrun/internal/qemu"
	"github.com/aibor/virtrun/internal/sys"
	"github.com/aibor/virtrun/internal/virtrun"
	"github.com/stretchr/testify/require"
)

//nolint:gochecknoglobals
var (
	KernelPath            = virtrun.FilePath("/kernels/vmlinuz")
	KernelArch            = sys.Native
	ForceTransportTypePCI bool
	Verbose               bool
)

//nolint:gochecknoinits
func init() {
	flag.TextVar(
		&KernelPath,
		"kernel.path",
		KernelPath,
		"absolute path of the test kernel",
	)
	flag.TextVar(
		&KernelArch,
		"kernel.arch",
		KernelArch,
		"architecture of the kernel",
	)
	flag.BoolVar(
		&ForceTransportTypePCI,
		"force-pci",
		ForceTransportTypePCI,
		"force transport type virtio-pci instead of arch default",
	)
	flag.BoolVar(
		&Verbose,
		"verbose",
		Verbose,
		"show complete guest output",
	)
}

func TestIntegration(t *testing.T) {
	t.Parallel()

	verboseFlag := func() string {
		if Verbose {
			return "-test.v"
		}

		return ""
	}

	tests := []struct {
		name       string
		bin        string
		args       []string
		standalone bool
		requireErr require.ErrorAssertionFunc
	}{
		{
			name:       "return 0",
			bin:        "bin/return",
			args:       []string{"0"},
			requireErr: require.NoError,
		},
		{
			name: "return 55",
			bin:  "bin/return",
			args: []string{"55"},
			requireErr: func(t require.TestingT, err error, _ ...any) {
				var qemuErr *qemu.CommandError

				require.ErrorAs(t, err, &qemuErr)
				require.Equal(t, 55, qemuErr.ExitCode)
			},
		},
		{
			name:       "panic",
			bin:        "bin/panic",
			standalone: true,
			requireErr: func(t require.TestingT, err error, _ ...any) {
				require.ErrorIs(t, err, qemu.ErrGuestPanic)
			},
		},
		{
			name: "oom",
			bin:  "bin/oom",
			args: []string{"128"},
			requireErr: func(t require.TestingT, err error, _ ...any) {
				require.ErrorIs(t, err, qemu.ErrGuestOom)
			},
		},
		{
			name: "linked",
			bin:  "../internal/initramfs/testdata/bin/main",
			requireErr: func(t require.TestingT, err error, _ ...any) {
				var qemuErr *qemu.CommandError

				require.ErrorAs(t, err, &qemuErr)

				expected := 73
				if !KernelArch.IsNative() {
					expected = 126
				}

				require.Equal(t, expected, qemuErr.ExitCode)
			},
		},
		{
			name: "guest test",
			bin:  "bin/guest.test",
			args: []string{
				verboseFlag(),
			},
			requireErr: require.NoError,
		},
		{
			name:       "guest standalone test",
			bin:        "bin/guest.standalone.test",
			standalone: true,
			args: []string{
				verboseFlag(),
				"-test.gocoverdir=/tmp/",
				"-test.coverprofile=/tmp/cover.out",
			},
			requireErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			binary, err := virtrun.AbsoluteFilePath(tt.bin)
			require.NoError(t, err)

			config, err := virtrun.New(KernelArch)
			require.NoError(t, err)

			config.Qemu.Kernel = KernelPath
			config.Qemu.Verbose = Verbose
			config.Qemu.Memory.Value = 128
			config.Qemu.InitArgs = tt.args
			config.Initramfs.Binary = binary
			config.Initramfs.StandaloneInit = tt.standalone

			if ForceTransportTypePCI {
				config.Qemu.TransportType = qemu.TransportTypePCI
			}

			irfs, err := virtrun.NewInitramfsArchive(config.Initramfs)
			require.NoError(t, err)
			t.Cleanup(func() { _ = irfs.Cleanup() })

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			t.Cleanup(cancel)

			cmd, err := virtrun.NewQemuCommand(ctx, config.Qemu, irfs.Path)
			require.NoError(t, err)

			t.Log(cmd.String())

			var stdOut, stdErr bytes.Buffer

			err = cmd.Run(&stdOut, &stdErr)

			t.Log(stdOut.String())
			t.Log(stdErr.String())

			tt.requireErr(t, err)
		})
	}
}
