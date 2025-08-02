// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package qemu

import (
	"bytes"
	"fmt"
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
	return func(t assert.TestingT, s, contains any, msgAndArgs ...any) bool {
		args, ok := s.([]Argument)
		if !ok {
			t.Errorf("argument should be []Argument")
			return false
		}

		for _, arg := range args {
			if name != arg.name {
				continue
			}

			return assertion(t, arg.value, contains, msgAndArgs...)
		}

		t.Errorf("Argument %s not found", name)

		return false
	}
}

func TestCommandSpec_Arguments(t *testing.T) {
	tests := []struct {
		name   string
		spec   CommandSpec
		expect any
		assert assert.ComparisonAssertionFunc
	}{
		{
			name: "machine params",
			spec: CommandSpec{
				Machine: "pc4.2",
				CPU:     "8086",
				SMP:     23,
				Memory:  269,
			},
			expect: []Argument{
				UniqueArg("machine", "pc4.2"),
				UniqueArg("cpu", "8086"),
				UniqueArg("smp", "23"),
				UniqueArg("m", "269"),
			},
			assert: assert.Subset,
		},
		{
			name:   "yes-kvm",
			spec:   CommandSpec{},
			expect: UniqueArg("enable-kvm"),
			assert: assert.Contains,
		},
		{
			name: "no-kvm",
			spec: CommandSpec{
				NoKVM: true,
			},
			expect: UniqueArg("enable-kvm"),
			assert: assert.NotContains,
		},

		{
			name: "yes-verbose",
			spec: CommandSpec{
				Verbose: true,
			},
			expect: "quiet",
			assert: ArgumentValueAssertionFunc("append", assert.NotContains),
		},

		{
			name:   "no-verbose",
			spec:   CommandSpec{},
			expect: "quiet",
			assert: ArgumentValueAssertionFunc("append", assert.Contains),
		},
		{
			name: "init args",
			spec: CommandSpec{
				InitArgs: []string{
					"first",
					"second",
					"third",
				},
			},
			expect: " -- first second third",
			assert: ArgumentValueAssertionFunc("append", assert.Contains),
		},
		{
			name: "serial files virtio-mmio",
			spec: CommandSpec{
				AdditionalConsoles: []string{
					"/output/file1",
					"/output/file2",
				},
				TransportType: TransportTypeMMIO,
			},
			expect: []Argument{
				RepeatableArg("device", "virtio-serial-device,max_ports=8"),
				RepeatableArg("chardev", "stdio,id=stdio"),
				RepeatableArg("device", "virtconsole,chardev=stdio"),
				RepeatableArg("chardev", "file,id=con0,path=/dev/fd/3"),
				RepeatableArg("device", "virtconsole,chardev=con0"),
				RepeatableArg("chardev", "file,id=con1,path=/dev/fd/4"),
				RepeatableArg("device", "virtconsole,chardev=con1"),
			},
			assert: assert.Subset,
		},
		{
			name: "serial files virtio-pci",
			spec: CommandSpec{
				AdditionalConsoles: []string{
					"/output/file1",
					"/output/file2",
				},
				TransportType: TransportTypePCI,
			},
			expect: []Argument{
				RepeatableArg("device", "virtio-serial-pci,max_ports=8"),
				RepeatableArg("chardev", "stdio,id=stdio"),
				RepeatableArg("device", "virtconsole,chardev=stdio"),
				RepeatableArg("chardev", "file,id=con0,path=/dev/fd/3"),
				RepeatableArg("device", "virtconsole,chardev=con0"),
				RepeatableArg("chardev", "file,id=con1,path=/dev/fd/4"),
				RepeatableArg("device", "virtconsole,chardev=con1"),
			},
			assert: assert.Subset,
		},
		{
			name: "serial files isa-pci",
			spec: CommandSpec{
				AdditionalConsoles: []string{
					"/output/file1",
					"/output/file2",
				},
				TransportType: TransportTypeISA,
			},
			expect: []Argument{
				RepeatableArg("chardev", "stdio,id=stdio"),
				RepeatableArg("serial", "chardev:stdio"),
				RepeatableArg("chardev", "file,id=con0,path=/dev/fd/3"),
				RepeatableArg("serial", "chardev:con0"),
				RepeatableArg("chardev", "file,id=con1,path=/dev/fd/4"),
				RepeatableArg("serial", "chardev:con1"),
			},
			assert: assert.Subset,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.assert(t, tt.spec.arguments(), tt.expect)
		})
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
				ExitCodeParser:     exitCodeScan,
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
				consoleOutput: []string{"one"},
			},
			assertErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := NewCommand(tt.spec)
			tt.assertErr(t, err)

			if tt.expected != nil {
				assert.Equal(t, tt.expected.String(), actual.String())
				assert.Equal(t, tt.expected.consoleOutput, actual.consoleOutput)

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
	tempDir := t.TempDir()

	exitCodeScanner := func(line []byte) (int, bool) {
		var exitCode int
		_, err := fmt.Sscanf(string(line), "exit code: %d", &exitCode)

		return exitCode, err == nil
	}

	tests := []struct {
		name           string
		cmd            Command
		expectedStdout string
		expectedStderr string
		assertErr      require.ErrorAssertionFunc
	}{
		{
			name: "just success",
			cmd: Command{
				name: "echo",
				args: []string{"exit code: 0"},
				stdoutParser: stdoutParser{
					ExitCodeParser: exitCodeScanner,
				},
			},
			assertErr: require.NoError,
		},
		{
			name: "success with stdout",
			cmd: Command{
				name: "echo",
				args: []string{"some\nexit code: 0"},
				stdoutParser: stdoutParser{
					ExitCodeParser: exitCodeScanner,
				},
			},
			expectedStdout: "some\n",
			assertErr:      require.NoError,
		},
		{
			name: "success with stdout and stderr",
			cmd: Command{
				name: "sh",
				args: []string{
					"-c",
					"echo fail >&2; echo 'some\nexit code: 0'",
				},
				stdoutParser: stdoutParser{
					ExitCodeParser: exitCodeScanner,
				},
			},
			expectedStdout: "some\n",
			expectedStderr: "fail\n",
			assertErr:      require.NoError,
		},
		{
			name: "success with consoles",
			cmd: Command{
				name: "sh",
				args: []string{
					"-c",
					"echo foo >&3; echo fail >&2; echo 'some\nexit code: 0'",
				},
				stdoutParser: stdoutParser{
					ExitCodeParser: exitCodeScanner,
				},
				consoleOutput: []string{
					tempDir + "/out1",
				},
			},
			expectedStdout: "some\n",
			expectedStderr: "fail\n",
			assertErr:      require.NoError,
		},
		{
			name: "success but consoles no output",
			cmd: Command{
				name: "sh",
				args: []string{
					"-c",
					"echo foo >&3; echo fail >&2; echo 'some\nexit code: 0'",
				},
				stdoutParser: stdoutParser{
					ExitCodeParser: exitCodeScanner,
				},
				consoleOutput: []string{
					tempDir + "/out1",
					tempDir + "/out2",
				},
			},
			expectedStdout: "some\n",
			expectedStderr: "fail\n",
			assertErr: func(t require.TestingT, err error, args ...any) {
				require.ErrorIs(t, err, ErrConsoleNoOutput, args...)
			},
		},
		{
			name: "missing exit code",
			cmd: Command{
				name: "sh",
				args: []string{"-c", "echo foo >&3; echo exit"},
				stdoutParser: stdoutParser{
					ExitCodeParser: exitCodeScanner,
				},
				consoleOutput: []string{
					tempDir + "/out1",
				},
			},
			expectedStdout: "exit\n",
			assertErr: func(t require.TestingT, err error, args ...any) {
				require.ErrorIs(t, err, ErrGuestNoExitCodeFound, args...)
			},
		},
		{
			name: "fail with consoles",
			cmd: Command{
				name: "sh",
				args: []string{"-c", "echo foo >&3; echo exit code: 42"},
				stdoutParser: stdoutParser{
					ExitCodeParser: exitCodeScanner,
				},
				consoleOutput: []string{
					tempDir + "/out1",
				},
			},
			assertErr: func(t require.TestingT, err error, args ...any) {
				var cmdErr *CommandError
				require.ErrorAs(t, err, &cmdErr, args...)
				assert.Equal(t, 42, cmdErr.ExitCode, args...)
				assert.Equal(t, ErrGuestNonZeroExitCode, cmdErr.Err, args...)
			},
		},
		{
			name: "start error with consoles",
			cmd: Command{
				name: "nonexistingprogramthatdoesnotexistanywhere",
				stdoutParser: stdoutParser{
					ExitCodeParser: exitCodeScanner,
				},
				consoleOutput: []string{
					tempDir + "/out1",
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

			var outBuf, errBuf bytes.Buffer

			err := tt.cmd.Run(t.Context(), nil, &outBuf, &errBuf)
			tt.assertErr(t, err)

			assert.Equal(t, tt.expectedStdout, outBuf.String(), "stdout")
			assert.Equal(t, tt.expectedStderr, errBuf.String(), "stderr")
		})
	}
}
