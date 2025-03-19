// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

//go:generate env CGO_ENABLED=0 go build -v -trimpath -buildvcs=false -o testdata/bin/ ./testdata/cmd/...

package cmd_test

import (
	"bytes"
	"flag"
	"strings"
	"testing"

	"github.com/aibor/virtrun/internal/cmd"
	"github.com/aibor/virtrun/internal/qemu"
	"github.com/aibor/virtrun/internal/sys"
	"github.com/stretchr/testify/assert"
)

var (
	KernelPath            = "/kernels/vmlinuz"
	ForceTransportTypePCI bool
	Verbose               bool
)

func init() {
	flag.StringVar(
		&KernelPath,
		"virtrun.kernel",
		KernelPath,
		"path of the test kernel",
	)
	flag.BoolVar(
		&ForceTransportTypePCI,
		"virtrun.forcepci",
		ForceTransportTypePCI,
		"force transport type virtio-pci instead of arch default",
	)
	flag.BoolVar(
		&Verbose,
		"virtrun.verbose",
		Verbose,
		"show complete guest output",
	)
}

func TestIntegration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		bin              string
		args             []string
		initArgs         []string
		expectedExitCode int
		expectedStdOut   string
		expectedStdErr   string
	}{
		{
			name:             "return 0",
			bin:              "testdata/bin/return",
			initArgs:         []string{"0"},
			expectedExitCode: 0,
		},
		{
			name:             "return 55",
			bin:              "testdata/bin/return",
			initArgs:         []string{"55"},
			expectedExitCode: 55,
		},
		{
			name:             "panic",
			bin:              "testdata/bin/panic",
			args:             []string{"-standalone"},
			expectedExitCode: -1,
			expectedStdOut:   "Kernel panic",
			expectedStdErr:   qemu.ErrGuestPanic.Error(),
		},
		{
			name:             "oom",
			bin:              "testdata/bin/oom",
			initArgs:         []string{"128"},
			expectedExitCode: -1,
			expectedStdOut:   "Killed process",
			expectedStdErr:   qemu.ErrGuestOom.Error(),
		},
		{
			name:     "cputest",
			bin:      "testdata/bin/cputest",
			args:     []string{"-smp", "2"},
			initArgs: []string{"2"},
		},
		{
			name:             "linked",
			bin:              "../sys/testdata/bin/main",
			expectedExitCode: 73,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			args := []string{
				"test",
				"-kernel", sys.MustAbsolutePath(KernelPath),
				"-cpu", "max",
				"-memory", "128",
			}
			if Verbose {
				args = append(args, "-verbose")
			}

			if ForceTransportTypePCI {
				args = append(args, "-transport", string(qemu.TransportTypePCI))
			}

			args = append(args, tt.args...)
			args = append(args, sys.MustAbsolutePath(tt.bin))
			args = append(args, tt.initArgs...)

			var stdOut, stdErr bytes.Buffer

			exitCode := cmd.Run(args, nil, &stdOut, &stdErr)
			assert.Equal(t, tt.expectedExitCode, exitCode, "exit code")

			assertBufContains(t, stdOut, tt.expectedStdOut, "stdout")
			assertBufContains(t, stdErr, tt.expectedStdErr, "stderr")
		})
	}
}

func assertBufContains(
	t *testing.T,
	buf bytes.Buffer,
	expected string,
	scope string,
) {
	t.Helper()

	actual := strings.TrimSpace(buf.String())
	if actual != "" {
		t.Log(scope+":", actual)
	}

	assert.Contains(t, actual, expected, scope)
}
