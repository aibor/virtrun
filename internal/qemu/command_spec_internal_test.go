// SPDX-FileCopyrightText: 2026 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package qemu

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
			expect: " -- \"first\" \"second\" \"third\"",
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
				RepeatableArg("chardev", "file,id=con1,path=/output/file1"),
				RepeatableArg("device", "virtconsole,chardev=con1"),
				RepeatableArg("chardev", "file,id=con2,path=/output/file2"),
				RepeatableArg("device", "virtconsole,chardev=con2"),
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
				RepeatableArg("chardev", "file,id=con1,path=/output/file1"),
				RepeatableArg("device", "virtconsole,chardev=con1"),
				RepeatableArg("chardev", "file,id=con2,path=/output/file2"),
				RepeatableArg("device", "virtconsole,chardev=con2"),
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
				RepeatableArg("chardev", "file,id=con1,path=/output/file1"),
				RepeatableArg("serial", "chardev:con1"),
				RepeatableArg("chardev", "file,id=con2,path=/output/file2"),
				RepeatableArg("serial", "chardev:con2"),
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
