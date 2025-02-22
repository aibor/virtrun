// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package qemu

import (
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
				ExitCodeFmt:        "rrr",
			},
			expectedCmd: &Command{
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
				},
				stdoutParser: stdoutParser{
					ExitCodeFmt: "rrr",
					Verbose:     true,
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

			if tt.expectedCmd != nil {
				assert.Equal(t, tt.expectedCmd.String(), actual.String())
				assert.Equal(t, tt.expectedCmd.stdoutParser, actual.stdoutParser)
				assert.Equal(t, tt.expectedCmd.consoleOutput, actual.consoleOutput)
			}
		})
	}
}

func TestCommand_Run(t *testing.T) {
	tempDir := t.TempDir()

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
					ExitCodeFmt: "rc: %d",
				},
			},
			assertErr: require.NoError,
		},
		{
			name: "success with consoles",
			cmd: Command{
				name: "echo",
				args: []string{"rc: 0"},
				stdoutParser: stdoutParser{
					ExitCodeFmt: "rc: %d",
				},
				consoleOutput: []string{
					tempDir + "/out1",
					tempDir + "/out2",
					tempDir + "/out3",
					tempDir + "/out4",
				},
			},
			assertErr: require.NoError,
		},
		{
			name: "fail with consoles",
			cmd: Command{
				name: "echo",
				args: []string{"rc: 42"},
				stdoutParser: stdoutParser{
					ExitCodeFmt: "rc: %d",
				},
				consoleOutput: []string{
					tempDir + "/out1",
					tempDir + "/out2",
					tempDir + "/out3",
					tempDir + "/out4",
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
				consoleOutput: []string{
					tempDir + "/out1",
					tempDir + "/out2",
					tempDir + "/out3",
					tempDir + "/out4",
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
