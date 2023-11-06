package main

import (
	"testing"

	"github.com/aibor/pidonetest/internal/qemu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		expectedBinaries []string
		expectedInitArgs []string
	}{
		{
			name: "usual go test arguments",
			args: []string{
				"bin.test",
				"-test.paniconexit0",
				"-test.v=true",
				"-test.timeout=10m0s",
			},
			expectedBinaries: []string{
				"bin.test",
			},
			expectedInitArgs: []string{
				"-test.paniconexit0",
				"-test.v=true",
				"-test.timeout=10m0s",
			},
		},
		{
			name: "go test arguments modified",
			args: []string{
				"bin.test",
				"-test.paniconexit0",
				"-test.gocoverdir=/some/where",
				"-test.coverprofile=cover.out",
			},
			expectedBinaries: []string{
				"bin.test",
			},
			expectedInitArgs: []string{
				"-test.paniconexit0",
				"-test.gocoverdir=/tmp",
				"-test.coverprofile=/dev/ttyS2",
			},
		},
		{
			name: "go test arguments modification suppressed",
			args: []string{
				"bin.test",
				"-test.paniconexit0",
				"--",
				"-test.gocoverdir=/some/where",
				"-test.coverprofile=cover.out",
			},
			expectedBinaries: []string{
				"bin.test",
			},
			expectedInitArgs: []string{
				"-test.paniconexit0",
				"-test.gocoverdir=/some/where",
				"-test.coverprofile=cover.out",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var (
				binaries   []string
				qemuCmd    qemu.Command
				standalone bool
			)

			execArgs := append([]string{"self"}, tt.args...)
			err := parseArgs(execArgs, &binaries, &qemuCmd, &standalone)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedBinaries, binaries, "binaries should be as expected")
			assert.Equal(t, tt.expectedInitArgs, qemuCmd.InitArgs, "init args should be as expected")
		})
	}
}
