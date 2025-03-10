// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

//go:generate env CGO_ENABLED=0 go build -v -trimpath -buildvcs=false -o bin/ ./cmd/...

package integration_test

import (
	"bytes"
	"context"
	"flag"
	"testing"
	"time"

	"github.com/aibor/virtrun/internal/cmd"
	"github.com/aibor/virtrun/internal/qemu"
	"github.com/aibor/virtrun/internal/virtrun"
	"github.com/stretchr/testify/require"
)

var (
	KernelPath            = "/kernels/vmlinuz"
	ForceTransportTypePCI bool
	Verbose               bool
)

func init() {
	flag.Var(
		(*cmd.FilePath)(&KernelPath),
		"kernel.path",
		"absolute path of the test kernel",
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

	tests := []struct {
		name       string
		bin        string
		prepare    func(spec *virtrun.Spec)
		requireErr require.ErrorAssertionFunc
	}{
		{
			name: "return 0",
			bin:  "bin/return",
			prepare: func(spec *virtrun.Spec) {
				spec.Qemu.InitArgs = []string{"0"}
			},
			requireErr: require.NoError,
		},
		{
			name: "return 55",
			bin:  "bin/return",
			prepare: func(spec *virtrun.Spec) {
				spec.Qemu.InitArgs = []string{"55"}
			},
			requireErr: func(t require.TestingT, err error, _ ...any) {
				var qemuErr *qemu.CommandError

				require.ErrorAs(t, err, &qemuErr)
				require.Equal(t, 55, qemuErr.ExitCode)
			},
		},
		{
			name: "panic",
			bin:  "bin/panic",
			prepare: func(spec *virtrun.Spec) {
				spec.Initramfs.StandaloneInit = true
			},
			requireErr: func(t require.TestingT, err error, _ ...any) {
				require.ErrorIs(t, err, qemu.ErrGuestPanic)
			},
		},
		{
			name: "oom",
			bin:  "bin/oom",
			prepare: func(spec *virtrun.Spec) {
				spec.Qemu.InitArgs = []string{"128"}
			},
			requireErr: func(t require.TestingT, err error, _ ...any) {
				require.ErrorIs(t, err, qemu.ErrGuestOom)
			},
		},
		{
			name: "cputest",
			bin:  "bin/cputest",
			prepare: func(spec *virtrun.Spec) {
				spec.Qemu.SMP = 2
				spec.Qemu.InitArgs = []string{"2"}
			},
			requireErr: require.NoError,
		},
		{
			name: "linked",
			bin:  "../internal/sys/testdata/bin/main",
			requireErr: func(t require.TestingT, err error, _ ...any) {
				var qemuErr *qemu.CommandError
				require.ErrorAs(t, err, &qemuErr)
				require.Equal(t, 73, qemuErr.ExitCode)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			spec := &virtrun.Spec{
				Qemu: virtrun.Qemu{
					Kernel:  KernelPath,
					Verbose: Verbose,
					CPU:     "max",
					Memory:  128,
					SMP:     1,
				},
				Initramfs: virtrun.Initramfs{
					Binary: cmd.MustAbsoluteFilePath(t, tt.bin),
				},
			}

			if tt.prepare != nil {
				tt.prepare(spec)
			}

			if ForceTransportTypePCI {
				spec.Qemu.TransportType = qemu.TransportTypePCI
			}

			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			t.Cleanup(cancel)

			var stdOut, stdErr bytes.Buffer

			err := virtrun.Run(ctx, spec, nil, &stdOut, &stdErr)

			t.Log(stdOut.String())
			t.Log(stdErr.String())

			tt.requireErr(t, err)
		})
	}
}
