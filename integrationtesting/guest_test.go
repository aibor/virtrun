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
	virtrunRoot, err := filepath.Abs("..")
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
				"-kernel", string(KernelPath),
			}

			if Verbose {
				virtrunArgs = append(virtrunArgs, "-verbose")
			}

			for _, path := range KernelModules {
				virtrunArgs = append(virtrunArgs, "-addModule", path)
			}

			testTags := []string{
				"integration_guest",
			}

			if tt.standalone {
				virtrunArgs = append(virtrunArgs, "-standalone")
				testTags = append(testTags, "standalone")
			}

			execArgs := []string{
				// Unset GOARCH for the exec command as it needs to run as
				// native arch of the test host.
				"env",
				"GOARCH=",
				"go",
				"run",
				virtrunRoot,
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
				// Set GOARCH so the test binary is compiled with the correct
				// arch.
				"GOARCH="+KernelArch,
				// Although virtrun consume GOARCH, we need to set VIRTRUN_ARCH
				// her as well, because we call virtrun wrapped in the "go run"
				// above. For "go run" we need to unset GOARCH so it runs
				// with the required host arch. Because of this, we need to set
				// VIRTRUN_ARCH here as well to end up with the requested arch.
				"VIRTRUN_ARCH="+KernelArch,
				"VIRTRUN_ARGS="+strings.Join(virtrunArgs, " "),
			)

			out, err := cmd.CombinedOutput()
			if len(out) > 0 {
				t.Log(string(out))
			}

			assert.NoError(t, err)
		})
	}
}
