// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

//go:generate env CGO_ENABLED=0 go build -v -trimpath -buildvcs=false -o testdata/bin/ ./testdata/cmd/...

package cmd_test

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aibor/virtrun/internal/cmd"
	"github.com/aibor/virtrun/internal/qemu"
	"github.com/aibor/virtrun/internal/sys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration(t *testing.T) {
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
			expectedStdOut:   "exit code: 0",
			expectedExitCode: 0,
		},
		{
			name:             "return 55",
			bin:              "testdata/bin/return",
			initArgs:         []string{"55"},
			expectedStdOut:   "exit code: 55",
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
			self, err := os.Executable()
			require.NoError(t, err)

			args := []string{
				filepath.Base(self),
			}

			// Pass flags from the test invocation. Must start with "--" to
			// terminate flag parsing of the test binary itself. At least the
			// kernel path must be passed, e.g.  "-- -kernel /boot/vmlinuz".
			// Alternatively, VIRTRUN_ARGS environment variable can be used to
			// pass args.
			// Some flags may be overridden by the test cases, though.
			args = append(args, flag.Args()...)
			args = append(args, tt.args...)
			args = append(args, sys.MustAbsolutePath(tt.bin))
			args = append(args, tt.initArgs...)

			t.Logf("virtrun args: % s", args)

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
