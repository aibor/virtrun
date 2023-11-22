package virtrun_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aibor/virtrun"
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
			cmd := virtrun.Command{
				InitArgs: tt.inputArgs,
			}
			virtrun.ProcessGoTestFlags(&cmd)

			assert.Equal(t, tt.expectedArgs, cmd.InitArgs)
			assert.Equal(t, tt.expectedFiles, cmd.AdditionalConsoles)
		})
	}
}
