package main

import (
	"path/filepath"
	"testing"

	"github.com/aibor/virtrun/internal/qemu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseArgs(t *testing.T) {
	absBinPath, err := filepath.Abs("bin.test")
	require.NoError(t, err)

	tests := []struct {
		name     string
		args     []string
		expected config
		errMsg   string
	}{
		{
			name: "requires kernel",
			args: []string{
				"bin.test",
			},
			errMsg: "no kernel given",
		},
		{
			name: "requires binary",
			args: []string{
				"-kernel=/boot/this",
			},
			errMsg: "no binary given",
		},
		{
			name: "simple go test invocation",
			args: []string{
				"-kernel=/boot/this",
				"bin.test",
				"-test.paniconexit0",
				"-test.v=true",
				"-test.timeout=10m0s",
			},
			expected: config{
				binaries: []string{
					absBinPath,
				},
				qemuCmd: qemu.Command{
					Kernel: "/boot/this",
					InitArgs: []string{
						"-test.paniconexit0",
						"-test.v=true",
						"-test.timeout=10m0s",
					},
				},
			},
		},
		{
			name: "go test invocation with virtrun flags",
			args: []string{
				"-kernel=/boot/this",
				"-cpu", "host",
				"-machine=pc",
				"-transport", "2",
				"-memory=269",
				"-verbose",
				"-smp", "7",
				"-nokvm=true",
				"-standalone",
				"bin.test",
				"-test.paniconexit0",
				"-test.v=true",
				"-test.timeout=10m0s",
			},
			expected: config{
				binaries: []string{
					absBinPath,
				},
				qemuCmd: qemu.Command{
					Kernel:        "/boot/this",
					CPU:           "host",
					Machine:       "pc",
					TransportType: qemu.TransportTypeMMIO,
					Memory:        269,
					Verbose:       true,
					NoKVM:         true,
					SMP:           7,
					InitArgs: []string{
						"-test.paniconexit0",
						"-test.v=true",
						"-test.timeout=10m0s",
					},
				},
				standalone: true,
			},
		},
		{
			name: "flag parsing stops at flags after positional arguments",
			args: []string{
				"-kernel=/boot/this",
				"bin.test",
				"-test.paniconexit0",
				"another.binary",
				"-x",
				"-standalone",
			},
			expected: config{
				binaries: []string{
					absBinPath,
				},
				qemuCmd: qemu.Command{
					Kernel: "/boot/this",
					InitArgs: []string{
						"-test.paniconexit0",
						"another.binary",
						"-x",
						"-standalone",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var cfg config

			execArgs := append([]string{"self"}, tt.args...)
			err := cfg.parseArgs(execArgs)

			if tt.errMsg != "" {
				assert.ErrorContains(t, err, tt.errMsg)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, cfg)
		})
	}
}
