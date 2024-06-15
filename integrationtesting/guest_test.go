// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

//go:build integration

package integrationtesting_test

import (
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

	tests := []struct {
		name       string
		standalone bool
	}{
		{
			name:       "wrapped",
			standalone: false,
		},
		{
			name:       "standalone",
			standalone: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			virtrunArgs := []string{
				"-kernel", KernelPath,
			}
			if Verbose {
				virtrunArgs = append(virtrunArgs, "-verbose")
			}

			testTags := []string{
				"integration_guest",
			}

			if tt.standalone {
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
				"GOARCH="+KernelArch,
				"VIRTRUN_ARCH="+KernelArch,
				"VIRTRUN_ARGS="+virtrunArgString,
			)

			out, err := cmd.CombinedOutput()
			if len(out) > 0 {
				t.Log(string(out))
			}

			assert.NoError(t, err)
		})
	}
}
