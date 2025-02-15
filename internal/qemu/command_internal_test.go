// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package qemu

import (
	"fmt"
	"strconv"
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

func TestCommandSpec_StaticArguments(t *testing.T) {
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
			},
			assert: assert.Subset,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.assert(t, tt.spec.staticArguments(), tt.expect)
		})
	}
}

func TestNewCommand(t *testing.T) {
	exitCodeScan := func(_ string) (int, error) { return 0, nil }

	tests := []struct {
		name        string
		spec        CommandSpec
		expectedCmd *Command
		assertErr   require.ErrorAssertionFunc
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
				ExitCodeScanFunc:   exitCodeScan,
			},
			expectedCmd: &Command{
				name: "test",
				args: []string{
					"-kernel",
					"-initrd",
					"-chardev", "stdio,id=stdio",
					"-serial", "chardev:stdio",
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
					"-chardev", "socket,id=socket0,path=one,abstract=on",
					"-serial", "chardev:socket0",
				},
				stdoutParser: stdoutParser{
					ExitCodeScan: exitCodeScan,
					Verbose:      true,
				},
				consoleOutput: map[string]string{"one": "one"},
			},
			assertErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hashFn := func(s string) string {
				return s
			}

			actual, err := NewCommand(tt.spec, hashFn)
			tt.assertErr(t, err)

			if tt.expectedCmd != nil {
				assert.Equal(t, tt.expectedCmd.String(), actual.String())
				assert.Equal(t, tt.expectedCmd.consoleOutput, actual.consoleOutput)

				// Hack: Compare string representations because functions can
				// not compared. The string representation has the address of
				// the function which is sufficient for our test case.
				assert.Equal(t, fmt.Sprintf("%v", tt.expectedCmd.stdoutParser),
					fmt.Sprintf("%v", actual.stdoutParser))
			}
		})
	}
}

func TestCommand_Run(t *testing.T) {
	tempDir := t.TempDir()

	exitCodeScanner := func(s string) (int, error) {
		d, found := strings.CutPrefix(s, "exit code: ")
		if !found {
			return 0, assert.AnError
		}

		return strconv.Atoi(d)
	}

	tests := []struct {
		name      string
		cmd       Command
		assertErr require.ErrorAssertionFunc
	}{
		{
			name: "success no consoles",
			cmd: Command{
				name: "echo",
				args: []string{"rc: 0"},
				stdoutParser: stdoutParser{
					ExitCodeScan: func(s string) (int, error) {
						if s != "rc: 0" {
							return 0, assert.AnError
						}

						return 0, nil
					},
				},
			},

			assertErr: require.NoError,
		},
		{
			name: "success with consoles",
			cmd: Command{
				name: "echo",
				args: []string{"exit code: 0"},
				stdoutParser: stdoutParser{
					ExitCodeScan: exitCodeScanner,
				},
				consoleOutput: map[string]string{
					"test-1-1": tempDir + "/out1",
					"test-1-2": tempDir + "/out2",
				},
			},
			assertErr: require.NoError,
		},
		{
			name: "fail with consoles",
			cmd: Command{
				name: "echo",
				args: []string{"exit code: 42"},
				stdoutParser: stdoutParser{
					ExitCodeScan: exitCodeScanner,
				},
				consoleOutput: map[string]string{
					"test-2-1": tempDir + "/out1",
					"test-2-2": tempDir + "/out2",
				},
			},
			assertErr: func(t require.TestingT, err error, _ ...any) {
				var cmdErr *CommandError
				require.ErrorAs(t, err, &cmdErr)
				assert.Equal(t, 42, cmdErr.ExitCode)
				assert.Equal(t, ErrGuestNonZeroExitCode, cmdErr.Err)
			},
		},
		{
			name: "start error with consoles",
			cmd: Command{
				name: "nonexistingprogramthatdoesnotexistanywhere",
				consoleOutput: map[string]string{
					"test-3-1": tempDir + "/out1",
					"test-3-2": tempDir + "/out2",
				},
			},
			assertErr: func(t require.TestingT, err error, _ ...any) {
				require.NotErrorIs(t, err, &CommandError{})
				require.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer goleak.VerifyNone(t)

			err := tt.cmd.Run(t.Context(), nil, nil, nil)
			tt.assertErr(t, err)
		})
	}
}
