// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package qemu

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

// ArgumentValueAssertionFunc returns an [assert.ComparisonAssertionFunc] that
// can be used to assert the value of the Argument with the given name.
func ArgumentValueAssertionFunc(
	name string,
	assertion assert.ComparisonAssertionFunc,
) assert.ComparisonAssertionFunc {
	return func(tt assert.TestingT, s, contains any, msgAndArgs ...any) bool {
		args, ok := s.([]Argument)
		if !ok {
			tt.Errorf("argument should be []Argument")
			return false
		}

		for _, arg := range args {
			if name != arg.name {
				continue
			}

			return assertion(tt, arg.value, contains, msgAndArgs...)
		}

		tt.Errorf("Argument %s not found", name)

		return false
	}
}

func TestNewCommand(t *testing.T) {
	exitCodeScan := func(_ []byte) (int, bool) { return 0, true }

	tests := []struct {
		name      string
		spec      CommandSpec
		expected  *Command
		assertErr require.ErrorAssertionFunc
	}{
		{
			name: "invalid",
			assertErr: func(t require.TestingT, err error, _ ...any) {
				require.ErrorIs(t, err, &ArgumentError{})
			},
		},
		{
			name: "with consoles",
			spec: CommandSpec{
				Executable:         "test",
				TransportType:      TransportTypeISA,
				AdditionalConsoles: []string{"one"},
				NoKVM:              true,
				Verbose:            true,
			},
			expected: &Command{
				name: "test",
				args: []string{
					"-kernel",
					"-initrd",
					"-chardev", "stdio,id=stdio",
					"-serial", "chardev:stdio",
					"-chardev", "file,id=con0,path=/dev/fd/3",
					"-serial", "chardev:con0",
					"-chardev", "file,id=con1,path=one",
					"-serial", "chardev:con1",
					"-display", "none",
					"-monitor", "none",
					"-no-reboot",
					"-nodefaults",
					"-no-user-config",
					"-append",
					"console=ttyS0",
					"panic=-1",
					"mitigations=off",
					"initcall_blacklist=ahci_pci_driver_init",
					"debug",
				},
				stdoutParser: stdoutParser{
					ExitCodeParser: exitCodeScan,
					Verbose:        true,
				},
			},
			assertErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := NewCommand(tt.spec, exitCodeScan)
			tt.assertErr(t, err)

			if tt.expected != nil {
				assert.Equal(t, tt.expected.String(), actual.String())

				// Hack: Compare string representations because functions can
				// not compared. The string representation has the address of
				// the function which is sufficient for our test case.
				assert.Equal(t, fmt.Sprintf("%v", tt.expected.stdoutParser),
					fmt.Sprintf("%v", actual.stdoutParser))
			}
		})
	}
}

func TestCommand_Run(t *testing.T) {
	exitCodeFmt := "exit code: %d"
	exitCode := func(e int) string {
		return fmt.Sprintf(exitCodeFmt, e)
	}
	exitCodeScanner := func(line []byte) (int, bool) {
		var d int

		_, err := fmt.Sscanf(string(line), exitCodeFmt, &d)

		return d, err == nil
	}

	tests := []struct {
		name           string
		cmd            Command
		expectedStdout []string
		expectedStderr []string
		assertErr      require.ErrorAssertionFunc
	}{
		{
			name: "just success",
			cmd: Command{
				name: "echo",
				args: []string{exitCode(0)},
				stdoutParser: stdoutParser{
					ExitCodeParser: exitCodeScanner,
				},
			},
			assertErr: require.NoError,
		},
		{
			name: "success with stderr",
			cmd: Command{
				name: "echo",
				args: []string{"some\n" + exitCode(0)},
				stdoutParser: stdoutParser{
					ExitCodeParser: exitCodeScanner,
				},
			},
			expectedStderr: []string{"some\n"},
			assertErr:      require.NoError,
		},
		{
			name: "success with stdout and stderr",
			cmd: Command{
				name: "sh",
				args: []string{
					"-c",
					"echo foo >&3;" +
						"echo fail >&2;" +
						"echo some;" +
						"echo " + exitCode(0),
				},
				stdoutParser: stdoutParser{
					ExitCodeParser: exitCodeScanner,
				},
			},
			expectedStdout: []string{"foo\n"},
			expectedStderr: []string{"some\n", "fail\n"},
			assertErr:      require.NoError,
		},
		{
			name: "missing exit code",
			cmd: Command{
				name: "sh",
				args: []string{"-c", "echo foo >&3; echo exit"},
				stdoutParser: stdoutParser{
					ExitCodeParser: exitCodeScanner,
				},
			},
			expectedStdout: []string{"foo\n"},
			expectedStderr: []string{"exit\n"},
			assertErr: func(t require.TestingT, err error, args ...any) {
				require.ErrorIs(t, err, ErrGuestNoExitCodeFound, args...)
			},
		},
		{
			name: "fail with consoles",
			cmd: Command{
				name: "sh",
				args: []string{
					"-c",
					"echo foo >&3; echo " + exitCode(42),
				},
				stdoutParser: stdoutParser{
					ExitCodeParser: exitCodeScanner,
				},
			},
			expectedStdout: []string{"foo\n"},
			assertErr: func(t require.TestingT, err error, args ...any) {
				var cmdErr *CommandError
				require.ErrorAs(t, err, &cmdErr, args...)
				assert.Equal(t, 42, cmdErr.ExitCode, args...)
				assert.Equal(t, ErrGuestNonZeroExitCode, cmdErr.Err, args...)
			},
		},
		{
			name: "start error",
			cmd: Command{
				name: "nonexistingprogramthatdoesnotexistanywhere",
				stdoutParser: stdoutParser{
					ExitCodeParser: exitCodeScanner,
				},
			},
			assertErr: func(t require.TestingT, err error, args ...any) {
				require.NotErrorIs(t, err, &CommandError{}, args...)
				require.Error(t, err, args...)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer goleak.VerifyNone(t)

			stdoutFile, err := os.CreateTemp(t.TempDir(), "stdout")
			require.NoError(t, err)

			stderrFile, err := os.CreateTemp(t.TempDir(), "stderr")
			require.NoError(t, err)

			err = tt.cmd.Run(t.Context(), nil, stdoutFile, stderrFile)
			tt.assertErr(t, err)

			assertLines(t, stdoutFile, tt.expectedStdout, "stdout")
			assertLines(t, stderrFile, tt.expectedStderr, "stderr")
		})
	}
}

func assertLines(t *testing.T, file *os.File, expected []string, args ...any) {
	t.Helper()

	_, err := file.Seek(0, 0)
	require.NoError(t, err, args...)

	output, err := io.ReadAll(file)
	require.NoError(t, err, args...)

	t.Logf("output: %q", string(output))

	stdoutLines := slices.Collect(strings.Lines(string(output)))
	assert.ElementsMatch(t, expected, stdoutLines, args...)
}
