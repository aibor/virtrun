// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package qemu_test

import (
	"testing"

	"github.com/aibor/virtrun/internal/pipe"
	"github.com/aibor/virtrun/internal/qemu"
	"github.com/stretchr/testify/assert"
)

func TestRewriteGoTestFlags(t *testing.T) {
	tests := []struct {
		name     string
		spec     qemu.CommandSpec
		expected qemu.CommandSpec
	}{
		{
			name: "empty",
		},
		{
			name: "usual go test flags",
			spec: qemu.CommandSpec{
				InitArgs: []string{
					"-test.paniconexit0",
					"-test.v=true",
					"-test.timeout=10m0s",
				},
			},
			expected: qemu.CommandSpec{
				InitArgs: []string{
					"-test.paniconexit0",
					"-test.v=true",
					"-test.timeout=10m0s",
				},
			},
		},
		{
			name: "go coverage flags",
			spec: qemu.CommandSpec{
				InitArgs: []string{
					"-test.paniconexit0",
					"-test.gocoverdir=/some/where",
					"-test.coverprofile=cover.out",
				},
			},
			expected: qemu.CommandSpec{
				InitArgs: []string{
					"-test.paniconexit0",
					"-test.gocoverdir=/tmp",
					"-test.coverprofile=" + pipe.Path(2),
				},
				AdditionalConsoles: []string{
					"cover.out",
				},
			},
		},
		{
			name: "go output dir dependent flags",
			spec: qemu.CommandSpec{
				InitArgs: []string{
					"-test.paniconexit0",
					"-test.blockprofile=block.out",
					"-test.cpuprofile=cpu.out",
					"-test.memprofile=mem.out",
					"-test.mutexprofile=mutex.out",
					"-test.trace=trace.out",
					"-test.outputdir=outputdir",
				},
			},
			expected: qemu.CommandSpec{
				InitArgs: []string{
					"-test.paniconexit0",
					"-test.blockprofile=" + pipe.Path(2),
					"-test.cpuprofile=" + pipe.Path(3),
					"-test.memprofile=" + pipe.Path(4),
					"-test.mutexprofile=" + pipe.Path(5),
					"-test.trace=" + pipe.Path(6),
					"-test.outputdir=/tmp",
				},
				AdditionalConsoles: []string{
					"outputdir/block.out",
					"outputdir/cpu.out",
					"outputdir/mem.out",
					"outputdir/mutex.out",
					"outputdir/trace.out",
				},
			},
		},
		{
			name: "go output dir dependent flags absolute paths",
			spec: qemu.CommandSpec{
				InitArgs: []string{
					"-test.paniconexit0",
					"-test.blockprofile=/tmp/block.out",
					"-test.cpuprofile=/tmp/cpu.out",
					"-test.memprofile=/tmp/mem.out",
					"-test.mutexprofile=/tmp/mutex.out",
					"-test.trace=/tmp/trace.out",
					"-test.outputdir=outputdir",
				},
			},
			expected: qemu.CommandSpec{
				InitArgs: []string{
					"-test.paniconexit0",
					"-test.blockprofile=" + pipe.Path(2),
					"-test.cpuprofile=" + pipe.Path(3),
					"-test.memprofile=" + pipe.Path(4),
					"-test.mutexprofile=" + pipe.Path(5),
					"-test.trace=" + pipe.Path(6),
					"-test.outputdir=/tmp",
				},
				AdditionalConsoles: []string{
					"/tmp/block.out",
					"/tmp/cpu.out",
					"/tmp/mem.out",
					"/tmp/mutex.out",
					"/tmp/trace.out",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.spec
			actual.RewriteGoTestFlagsPath()
			assert.Equal(t, tt.expected, actual)
		})
	}
}
