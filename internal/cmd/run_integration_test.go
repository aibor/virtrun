// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

//go:generate env CGO_ENABLED=0 go build -v -trimpath -buildvcs=false -o testdata/bin/ ./testdata/cmd/...

package cmd_test

import (
	"bytes"
	"flag"
	"io"
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

type outputAssertionFunc func(t *testing.T, file *os.File, args ...any)

func TestIntegration(t *testing.T) {
	tests := []struct {
		name             string
		bin              string
		args             []string
		initArgs         []string
		expectedExitCode int
		assertStdout     outputAssertionFunc
		assertStderr     outputAssertionFunc
	}{
		{
			name:             "return 0",
			bin:              "testdata/bin/return",
			initArgs:         []string{"0"},
			assertStdout:     assertOutputIs("exit code: 0"),
			expectedExitCode: 0,
		},
		{
			name:             "return 55",
			bin:              "testdata/bin/return",
			initArgs:         []string{"55"},
			assertStdout:     assertOutputIs("exit code: 55"),
			expectedExitCode: 55,
		},
		{
			name:         "output all bytes",
			bin:          "testdata/bin/output",
			initArgs:     []string{"256", "1"},
			assertStdout: assertOutputIs(makeOutput(256, 1)),
		},
		{
			name:         "output all bytes multi line",
			bin:          "testdata/bin/output",
			initArgs:     []string{"65536", "500"},
			assertStdout: assertOutputIs(makeOutput(65536, 500)),
		},
		{
			name:         "output all bytes long line",
			bin:          "testdata/bin/output",
			initArgs:     []string{"33554432", "1"},
			assertStdout: assertOutputIs(makeOutput(1<<25, 1)),
		},
		{
			name:             "panic",
			bin:              "testdata/bin/panic",
			args:             []string{"-standalone"},
			expectedExitCode: -1,
			assertStderr:     assertOutputContains(qemu.ErrGuestPanic.Error()),
		},
		{
			name:             "oom",
			bin:              "testdata/bin/oom",
			initArgs:         []string{"128"},
			expectedExitCode: -1,
			assertStderr:     assertOutputContains(qemu.ErrGuestOom.Error()),
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

			stdOutFile, err := os.CreateTemp(t.TempDir(), "stdout")
			require.NoError(t, err)

			stdErrFile, err := os.CreateTemp(t.TempDir(), "stderr")
			require.NoError(t, err)

			exitCode := cmd.Run(t.Context(), args, cmd.IO{
				Stdout: stdOutFile,
				Stderr: stdErrFile,
			})

			assert.Equal(t, tt.expectedExitCode, exitCode, "exit code")

			if tt.assertStdout != nil {
				tt.assertStdout(t, stdOutFile, "stdout")
			}

			if tt.assertStderr != nil {
				tt.assertStderr(t, stdErrFile, "stderr")
			}
		})
	}
}

func assertOutputContains(expected string) outputAssertionFunc {
	return func(t *testing.T, file *os.File, args ...any) {
		t.Helper()

		actual := readOutputFile(t, file, args...)
		assert.Contains(t, actual, expected, args...)
	}
}

func assertOutputIs(expected string) outputAssertionFunc {
	return func(t *testing.T, file *os.File, args ...any) {
		t.Helper()

		actual := readOutputFile(t, file, args...)
		if assert.Len(t, actual, len(expected), "length") {
			assert.Equal(t, expected, actual, args...)
		}
	}
}

func readOutputFile(t *testing.T, file *os.File, args ...any) string {
	t.Helper()

	_, err := file.Seek(0, 0)
	require.NoError(t, err, args...)

	stdOut, err := io.ReadAll(file)
	require.NoError(t, err, args...)

	return strings.TrimSpace(string(stdOut))
}

func makeOutput(length int, lines int) string {
	const maxBytes = 256

	line := make([]byte, length)
	for i := range length {
		line[i] = byte(i % maxBytes)
	}

	var output bytes.Buffer

	for i := range lines {
		if i > 0 {
			_ = output.WriteByte('\n')
		}

		_, _ = output.Write(line)
	}

	return output.String()
}
