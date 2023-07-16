package pidonetest_test

import (
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
		"/sys/kernel/tracing",
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
	cmd := exec.Command("./init")
	require.NoError(t, cmd.Start(), "command must start")
	checkExitCode := func() bool {
		err := cmd.Wait()
		return err != nil && cmd.ProcessState.ExitCode() == 1
	}
	assert.Eventually(t, checkExitCode, 100*time.Millisecond, 10*time.Millisecond)
}
