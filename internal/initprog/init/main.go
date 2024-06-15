// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

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

func runInit() (int, error) {
	env := []string{
		// Set PATH environment variable to the directory all additional files
		// are written to by virtrun.
		"PATH=/data",
	}

	err := sysinit.Run(func() (int, error) {
		// "/main" is the file virtrun copies the given binary to.
		cmd := exec.Command("/main", os.Args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = append(cmd.Environ(), env...)

		return 0, cmd.Run()
	})
	if errors.Is(err, sysinit.ErrNotPidOne) {
		return 127, err //nolint:gomnd,mnd
	}

	return 126, err //nolint:gomnd,mnd
}

func main() {
	rc, err := runInit()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}

	os.Exit(rc)
}
