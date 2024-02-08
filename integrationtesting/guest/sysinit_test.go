//go:build integration_guest

package main

import (
	"context"
	"fmt"
	"net"
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

func TestLoopbackInterface(t *testing.T) {
	iface, err := net.InterfaceByName("lo")
	require.NoError(t, err, "must get interface")

	assert.True(t, iface.Flags&net.FlagUp > 0)

	addrs, err := iface.Addrs()
	require.NoError(t, err, "must get addresses")

	assert.Len(t, addrs, 2, "should have 2 addresses")

	assert.Equal(t, addrs[0].String(), "127.0.0.1/8")
	assert.Equal(t, addrs[1].String(), "::1/128")
}

func TestEnv(t *testing.T) {
	if os.Getpid() == 1 {
		t.Skip("env only tested when called by default init")
	}
	envPath, envPathExists := os.LookupEnv("PATH")
	if assert.True(t, envPathExists, "PATH env var should be present") {
		assert.Equal(t, "/data", envPath, "PATH env var should be correct")
	}
}

func TestCommonSymlinks(t *testing.T) {
	symlinks := map[string]string{
		"/dev/fd":     "/proc/self/fd/",
		"/dev/stdin":  "/proc/self/fd/0",
		"/dev/stdout": "/proc/self/fd/1",
		"/dev/stderr": "/proc/self/fd/2",
	}

	for link, expectedTarget := range symlinks {
		t.Run(link, func(t *testing.T) {
			target, err := os.Readlink(link)
			require.NoError(t, err, "link must be readable")
			assert.Equal(t, expectedTarget, target, "link target should be as expected")
		})
	}
}
