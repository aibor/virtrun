// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package qemu_test

import (
	"testing"

	"github.com/aibor/virtrun/internal/qemu"
	"github.com/stretchr/testify/assert"
)

func TestCommmandConsoleDeviceName(t *testing.T) {
	tests := []struct {
		id        uint8
		transport qemu.TransportType
		console   string
	}{
		{
			id:        5,
			transport: qemu.TransportTypeISA,
			console:   "ttyS5",
		},
		{
			id:        3,
			transport: qemu.TransportTypePCI,
			console:   "hvc3",
		},
		{
			id:        1,
			transport: qemu.TransportTypeMMIO,
			console:   "hvc1",
		},
	}
	for _, tt := range tests {
		s := qemu.Command{
			TransportType: tt.transport,
		}
		assert.Equal(t, tt.console, s.TransportType.ConsoleDeviceName(tt.id))
	}
}

func TestCommmandAddExtraFile(t *testing.T) {
	s := qemu.Command{}
	d1 := s.AddConsole("test")
	d2 := s.AddConsole("real")
	assert.Equal(t, "ttyS1", d1)
	assert.Equal(t, "ttyS2", d2)
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
				"-test.coverprofile=/dev/ttyS1",
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
				"-test.blockprofile=/dev/ttyS1",
				"-test.cpuprofile=/dev/ttyS2",
				"-test.memprofile=/dev/ttyS3",
				"-test.mutexprofile=/dev/ttyS4",
				"-test.trace=/dev/ttyS5",
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			cmd := qemu.Command{
				InitArgs: tt.inputArgs,
			}
			cmd.ProcessGoTestFlags()

			assert.Equal(t, tt.expectedArgs, cmd.InitArgs)
			assert.Equal(t, tt.expectedFiles, cmd.AdditionalConsoles)
		})
	}
}
