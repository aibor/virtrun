// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

// Simple init program that can be pre-compiled for multiple architectures and
// embedded into the main binary.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/aibor/virtrun/sysinit"
)

func main() {
	cfg := sysinit.DefaultConfig()
	cfg.ModulesDir = "/lib/modules"
	// Set PATH environment variable to the directory all additional files
	// are written to by virtrun.
	cfg.Env["PATH"] = "/data"

	sysinit.Main(cfg, func() (int, error) {
		// "/main" is the file virtrun copies the given binary to.
		cmd := exec.Command("/main", os.Args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		var exitErr *exec.ExitError

		err := cmd.Run()
		if err != nil {
			if errors.As(err, &exitErr) {
				return exitErr.ExitCode(), nil
			}

			return -1, fmt.Errorf("main: %w", err)
		}

		return 0, nil
	})
}
