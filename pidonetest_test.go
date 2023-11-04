package pidonetest_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMountPoints(t *testing.T) {
	mountPoints := []string{
		"/dev",
		"/proc",
		"/run",
		"/sys",
		"/sys/fs/bpf",
		"/tmp",
	}

	mounts, err := os.ReadFile("/proc/mounts")
	require.NoError(t, err, "must read mounts file")
	t.Log("\n", string(mounts))

	for _, mp := range mountPoints {
		t.Run(mp, func(t *testing.T) {
			assert.Contains(t, string(mounts), fmt.Sprintf(" %s ", mp))
		})
	}
}

func TestNotPidOne(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	cmd := exec.CommandContext(ctx, "/init")
	require.NoError(t, cmd.Start(), "command must start")
	assert.Error(t, cmd.Wait(), "command should have exited with error")
	if assert.NotNil(t, cmd.ProcessState, "process state should be present") {
		assert.Equal(t, 127, cmd.ProcessState.ExitCode(), "exit code should be as expected")
	}
}
