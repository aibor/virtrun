// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

// Simple init program that can be pre-compiled for multiple architectures and
// embedded into the main binary.
package main

// Pre-compile init programs for all supported architectures. Statically linked
// so they can be used on any host platform.
//
//go:generate -command myenv env CGO_ENABLED=0 GOOS=linux
//go:generate myenv GOARCH=amd64 go build -buildvcs=false -trimpath -ldflags "-s -w" -o ../bin/amd64 .
//go:generate myenv GOARCH=arm64 go build -buildvcs=false -trimpath -ldflags "-s -w" -o ../bin/arm64 .
//go:generate myenv GOARCH=riscv64 go build -buildvcs=false -trimpath -ldflags "-s -w" -o ../bin/riscv64 .

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/aibor/virtrun/sysinit"
)

func run(mainFunc sysinit.Func) {
	log.SetFlags(log.Lmicroseconds)
	log.SetPrefix("VIRTRUN INIT: ")

	// Set PATH environment variable to the directory all additional files are
	// written to by virtrun.
	env := sysinit.EnvVars{"PATH": "/data"}

	sysinit.Run(
		sysinit.ExitCodePrinter(os.Stdout),
		sysinit.WithMountPoints(sysinit.SystemMountPoints()),
		sysinit.WithModules("/lib/modules/*"),
		sysinit.WithInterfaceUp("lo"),
		sysinit.WithSymlinks(sysinit.DevSymlinks()),
		sysinit.WithEnv(env),
		mainFunc,
	)
}

func main() {
	run(func() error {
		// "/main" is the file virtrun copies the given binary to.
		cmd := exec.Command("/main", os.Args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				err = sysinit.ExitError(exitErr.ExitCode())
			}

			return fmt.Errorf("main program: %w", err)
		}

		return nil
	})
}
