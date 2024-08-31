// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package qemu_test

import (
	"testing"

	"github.com/aibor/virtrun/internal/qemu"
	"github.com/stretchr/testify/assert"
)

func TestCommmandAddExtraFile(t *testing.T) {
	s := qemu.CommandSpec{}
	d1 := s.AddConsole("test")
	d2 := s.AddConsole("real")

	assert.Equal(t, "hvc1", d1)
	assert.Equal(t, "hvc2", d2)
	assert.Equal(t, []string{"test", "real"}, s.AdditionalConsoles)
}

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
				"-test.coverprofile=/dev/hvc1",
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
				"-test.blockprofile=/dev/hvc1",
				"-test.cpuprofile=/dev/hvc2",
				"-test.memprofile=/dev/hvc3",
				"-test.mutexprofile=/dev/hvc4",
				"-test.trace=/dev/hvc5",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := qemu.CommandSpec{
				InitArgs: tt.inputArgs,
			}
			cmd.ProcessGoTestFlags()

			assert.Equal(t, tt.expectedArgs, cmd.InitArgs)
			assert.Equal(t, tt.expectedFiles, cmd.AdditionalConsoles)
		})
	}
}
