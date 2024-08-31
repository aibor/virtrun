// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package qemu

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandArgs(t *testing.T) {
	tests := []struct {
		name   string
		cmd    CommandSpec
		expect any
		assert assert.ComparisonAssertionFunc
	}{
		{
			name: "machine params",
			cmd: CommandSpec{
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
			cmd:    CommandSpec{},
			expect: UniqueArg("enable-kvm"),
			assert: assert.Contains,
		},
		{
			name: "no-kvm",
			cmd: CommandSpec{
				NoKVM: true,
			},
			expect: UniqueArg("enable-kvm"),
			assert: assert.NotContains,
		},

		{
			name: "yes-verbose",
			cmd: CommandSpec{
				Verbose: true,
			},
			expect: "quiet",
			assert: ArgumentValueAssertionFunc("append", assert.NotContains),
		},

		{
			name:   "no-verbose",
			cmd:    CommandSpec{},
			expect: "quiet",
			assert: ArgumentValueAssertionFunc("append", assert.Contains),
		},
		{
			name: "init args",
			cmd: CommandSpec{
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
			cmd: CommandSpec{
				AdditionalConsoles: []string{
					"/output/file1",
					"/output/file2",
				},
				TransportType: TransportTypeMMIO,
			},
			expect: []Argument{
				RepeatableArg("chardev", "file,id=vcon1,path=/dev/fd/1"),
				RepeatableArg("chardev", "file,id=vcon3,path=/dev/fd/3"),
				RepeatableArg("chardev", "file,id=vcon4,path=/dev/fd/4"),
			},
			assert: assert.Subset,
		},
		{
			name: "serial files isa-pci",
			cmd: CommandSpec{
				AdditionalConsoles: []string{
					"/output/file1",
					"/output/file2",
				},
				TransportType: TransportTypeISA,
			},
			expect: []Argument{
				RepeatableArg("serial", "file:/dev/fd/1"),
				RepeatableArg("serial", "file:/dev/fd/3"),
				RepeatableArg("serial", "file:/dev/fd/4"),
			},
			assert: assert.Subset,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.assert(t, tt.cmd.arguments(), tt.expect)
		})
	}
}
