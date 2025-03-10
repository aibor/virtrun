// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration_guest

package main_test

import (
	"bufio"
	"context"
	"flag"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMountPoints(t *testing.T) {
	mounts := map[string]string{
		"/dev":                "devtmpfs",
		"/dev/shm":            "tmpfs",
		"/proc":               "proc",
		"/run":                "tmpfs",
		"/sys":                "sysfs",
		"/sys/fs/bpf":         "bpf",
		"/sys/kernel/debug":   "debugfs",
		"/sys/kernel/tracing": "tracefs",
		"/tmp":                "tmpfs",
	}

	mountsFile, err := os.ReadFile("/proc/mounts")
	require.NoError(t, err)

	actual := map[string]string{}

	scanner := bufio.NewScanner(strings.NewReader(string(mountsFile)))
	for scanner.Scan() {
		columns := strings.Fields(scanner.Text())
		actual[columns[1]] = columns[2]
	}

	require.NoError(t, scanner.Err(), "must read mounts file")

	for path, fsType := range mounts {
		t.Run(strings.ReplaceAll(path, "/", "_"), func(t *testing.T) {
			if assert.Contains(t, actual, path, "path should exist") {
				assert.Equal(t, fsType, actual[path], "type should match")
			}
		})
	}
}

func TestNotPidOne(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	t.Cleanup(cancel)

	cmd := exec.CommandContext(ctx, "/init")
	require.NoError(t, cmd.Start(), "command must start")
	require.Error(t, cmd.Wait(), "command should have exited with error")

	if assert.NotNil(t, cmd.ProcessState, "process state should be present") {
		actual := cmd.ProcessState.ExitCode()
		assert.Equal(t, 254, actual, "exit code should be as expected")
	}
}

func TestLoopbackInterface(t *testing.T) {
	iface, err := net.InterfaceByName("lo")
	require.NoError(t, err, "must get interface")

	assert.Positive(t, iface.Flags&net.FlagUp)

	addrs, err := iface.Addrs()
	require.NoError(t, err, "must get addresses")

	require.Len(t, addrs, 2, "should have 2 addresses")

	assert.Equal(t, "127.0.0.1/8", addrs[0].String())
	assert.Equal(t, "::1/128", addrs[1].String())
}

func TestEnv(t *testing.T) {
	envPath, envPathExists := os.LookupEnv("PATH")

	if assert.True(t, envPathExists, "PATH env var should be present") {
		assert.Equal(t, "/data", envPath, "PATH env var should be correct")
	}
}

func TestCommonSymlinks(t *testing.T) {
	symlinks := map[string]string{
		"/dev/core":   "/proc/kcore",
		"/dev/fd":     "/proc/self/fd/",
		"/dev/rtc":    "rtc0",
		"/dev/stdin":  "/proc/self/fd/0",
		"/dev/stdout": "/proc/self/fd/1",
		"/dev/stderr": "/proc/self/fd/2",
	}

	for link, expectedTarget := range symlinks {
		t.Run(strings.ReplaceAll(link, "/", "_"), func(t *testing.T) {
			target, err := os.Readlink(link)
			require.NoError(t, err, "link must be readable")
			assert.Equal(t, expectedTarget, target, "link target should be as expected")
		})
	}
}

var testModules = flag.String("testModules", "", "module names to test")

func TestModules(t *testing.T) {
	modules, err := os.ReadFile("/proc/modules")
	require.NoError(t, err)

	expected := []string{}
	if *testModules != "" {
		expected = strings.Split(*testModules, ",")
	}

	actual := []string{}

	scanner := bufio.NewScanner(strings.NewReader(string(modules)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		actual = append(actual, fields[0])
	}

	require.NoError(t, scanner.Err(), "must read modules file")

	t.Log("actual: ", actual)

	assert.ElementsMatch(t, actual, expected)
}

var testCPUs = flag.Int("cpus", 0, "cpus to expect")

func TestCPUs(t *testing.T) {
	expected := 1
	if *testCPUs > 0 {
		expected = *testCPUs
	}

	assert.Equal(t, expected, runtime.NumCPU())
}
