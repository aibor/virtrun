// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package virtrun

import (
	"testing"

	"github.com/aibor/virtrun/internal/pipe"
	"github.com/aibor/virtrun/internal/qemu"
	"github.com/stretchr/testify/assert"
)

func TestProcessGoTestFlags(t *testing.T) {
	tests := []struct {
		name          string
		inputArgs     []string
		expectedArgs  []string
		expectedFiles []string
	}{
		{
			name: "empty",
		},
		{
			name: "usual go test flags",
			inputArgs: []string{
				"-test.paniconexit0",
				"-test.v=true",
				"-test.timeout=10m0s",
			},
			expectedArgs: []string{
				"-test.paniconexit0",
				"-test.v=true",
				"-test.timeout=10m0s",
			},
		},
		{
			name: "go coverage flags",
			inputArgs: []string{
				"-test.paniconexit0",
				"-test.gocoverdir=/some/where",
				"-test.coverprofile=cover.out",
			},
			expectedArgs: []string{
				"-test.paniconexit0",
				"-test.gocoverdir=/tmp",
				"-test.coverprofile=" + pipe.Path(1),
			},
			expectedFiles: []string{
				"cover.out",
			},
		},
		{
			name: "go output dir dependent flags",
			inputArgs: []string{
				"-test.paniconexit0",
				"-test.blockprofile=block.out",
				"-test.cpuprofile=cpu.out",
				"-test.memprofile=mem.out",
				"-test.mutexprofile=mutex.out",
				"-test.trace=trace.out",
				"-test.outputdir=outputdir",
			},
			expectedArgs: []string{
				"-test.paniconexit0",
				"-test.blockprofile=" + pipe.Path(1),
				"-test.cpuprofile=" + pipe.Path(2),
				"-test.memprofile=" + pipe.Path(3),
				"-test.mutexprofile=" + pipe.Path(4),
				"-test.trace=" + pipe.Path(5),
				"-test.outputdir=/tmp",
			},
			expectedFiles: []string{
				"outputdir/block.out",
				"outputdir/cpu.out",
				"outputdir/mem.out",
				"outputdir/mutex.out",
				"outputdir/trace.out",
			},
		},
		{
			name: "go output dir dependent flags absolute paths",
			inputArgs: []string{
				"-test.paniconexit0",
				"-test.blockprofile=/tmp/block.out",
				"-test.cpuprofile=/tmp/cpu.out",
				"-test.memprofile=/tmp/mem.out",
				"-test.mutexprofile=/tmp/mutex.out",
				"-test.trace=/tmp/trace.out",
				"-test.outputdir=outputdir",
			},
			expectedArgs: []string{
				"-test.paniconexit0",
				"-test.blockprofile=" + pipe.Path(1),
				"-test.cpuprofile=" + pipe.Path(2),
				"-test.memprofile=" + pipe.Path(3),
				"-test.mutexprofile=" + pipe.Path(4),
				"-test.trace=" + pipe.Path(5),
				"-test.outputdir=/tmp",
			},
			expectedFiles: []string{
				"/tmp/block.out",
				"/tmp/cpu.out",
				"/tmp/mem.out",
				"/tmp/mutex.out",
				"/tmp/trace.out",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmdSpec := qemu.CommandSpec{
				InitArgs: tt.inputArgs,
			}
			rewriteGoTestFlagsPath(&cmdSpec)

			var actualFiles []string

			actualFiles = append(actualFiles, cmdSpec.AdditionalConsoles...)

			assert.Equal(t, tt.expectedArgs, cmdSpec.InitArgs)
			assert.Equal(t, tt.expectedFiles, actualFiles)
		})
	}
}
