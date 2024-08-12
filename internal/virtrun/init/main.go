// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

// Simple init program that can be pre-compiled for multiple architectures and
// embedded into the main binary.
package main

import (
	"errors"
	"os"
	"os/exec"

	"github.com/aibor/virtrun/sysinit"
)

func main() {
	env := []string{
		// Set PATH environment variable to the directory all additional files
		// are written to by virtrun.
		"PATH=/data",
	}

	cfg := sysinit.DefaultConfig()
	cfg.ModulesDir = "/lib/modules"

	err := sysinit.Run(cfg, func() (int, error) {
		// "/main" is the file virtrun copies the given binary to.
		cmd := exec.Command("/main", os.Args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = append(cmd.Environ(), env...)

		return 0, cmd.Run()
	})
	if err != nil {
		exitCode := 126
		if errors.Is(err, sysinit.ErrNotPidOne) {
			exitCode = 127
		}

		sysinit.PrintError(os.Stderr, err)

		os.Exit(exitCode)
	}
}
