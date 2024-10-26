// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sys_test

import (
	"os/exec"
	"testing"

	"github.com/aibor/virtrun/internal/sys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTestdata(t *testing.T) {
	var cmdErr *exec.ExitError

	cmd := exec.Command("testdata/bin/main")
	require.ErrorAs(t, cmd.Run(), &cmdErr)

	// 73 is the exit code of the test binary if everything is properly linked.
	assert.Equal(t, 73, cmdErr.ExitCode())
}

func TestFilesLdd(t *testing.T) {
	actual, err := sys.Ldd("testdata/bin/main")
	require.NoErrorf(t, err, "must resolve")

	expected := []string{
		"testdata/lib/libfunc2.so",
		"testdata/lib/libfunc3.so",
		"testdata/lib/libfunc1.so",
	}

	sys.AssertContainsPaths(t, actual, expected)
}
