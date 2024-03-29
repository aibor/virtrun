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
			virtrunArgs := []string{
				"-kernel", KernelPath,
			}
			if Verbose {
				virtrunArgs = append(virtrunArgs, "-verbose")
			}
			testTags := []string{
				"integration_guest",
			}

			if standalone {
				virtrunArgs = append(virtrunArgs, "-standalone")
				testTags = append(testTags, "standalone")
			}

			virtrunArgString := strings.Join(virtrunArgs, " ")
			// Unset GOARCH for the exec command as it needs to run as native
			// arch of the test host.
			execString := "env GOARCH= go run " + filepath.Join(cwd, "..")
			tagString := strings.Join(testTags, ",")

			args := []string{
				"test",
				"-v",
				"-timeout", "2m",
				"-exec", execString,
				"-tags", tagString,
				"-cover",
				"-coverprofile", "/tmp/cover.out",
				"-coverpkg", "github.com/aibor/virtrun/sysinit",
				"./guest/...",
			}

			cmd := exec.Command("go", args...)
			cmd.Env = append(
				os.Environ(),
				fmt.Sprintf("GOARCH=%s", KernelArch),
				fmt.Sprintf("VIRTRUN_ARCH=%s", KernelArch),
				fmt.Sprintf("VIRTRUN_ARGS=%s", virtrunArgString),
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
