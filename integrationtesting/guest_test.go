// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

//go:build integration

package integrationtesting

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGuestSysinit(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	test := func(standalone bool) func(t *testing.T) {
		return func(t *testing.T) {
			execArgs := []string{
				"env",
				"GOARCH=",
				fmt.Sprintf("QEMU_ARCH=%s", KernelArch),
				"go",
				"run",
				filepath.Join(cwd, ".."),
				"-kernel", KernelPath,
			}
			if Verbose {
				execArgs = append(execArgs, "-verbose")
			}
			testTags := []string{
				"integration_guest",
			}

			if standalone {
				execArgs = append(execArgs, "-standalone")
				testTags = append(testTags, "standalone")
			}
			args := []string{
				"test",
				"-v",
				"-timeout", "2m",
				"-exec", strings.Join(execArgs, " "),
				"-tags", strings.Join(testTags, ","),
				"-cover",
				"-coverprofile", "/tmp/cover.out",
				"-coverpkg", "github.com/aibor/virtrun/sysinit",
				"./guest/...",
			}

			cmd := exec.Command("go", args...)
			cmd.Env = append(
				os.Environ(),
				fmt.Sprintf("GOARCH=%s", KernelArch),
			)
			out, err := cmd.CombinedOutput()
			if len(out) > 0 {
				t.Log(string(out))
			}
			assert.NoError(t, err)
		}
	}

	t.Run("wrapped", test(false))
	t.Run("standalone", test(true))
}
