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

func run(mainFunc func() (int, error)) {
	log.SetFlags(log.Lmicroseconds)
	log.SetPrefix("VIRTRUN INIT: ")

	// Set PATH environment variable to the directory all additional files are
	// written to by virtrun.
	env := sysinit.EnvVars{"PATH": "/data"}

	sysinit.Run(
		sysinit.WithMountPoints(sysinit.SystemMountPoints()),
		sysinit.WithModules("/lib/modules/*"),
		sysinit.WithInterfaceUp("lo"),
		sysinit.WithSymlinks(sysinit.DevSymlinks()),
		sysinit.WithEnv(env),
		sysinit.WithHostPipes(),
		sysinit.WithStdoutHostPipe(),
		func(state *sysinit.State) error {
			exitCode, err := mainFunc()
			if err != nil {
				return err
			}

			state.SetExitCode(exitCode)

			return nil
		},
	)
}

func main() {
	run(func() (int, error) {
		// "/main" is the file virtrun copies the given binary to.
		cmd := exec.Command("/main", os.Args[1:]...) //nolint:noctx
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			var exitErr *exec.ExitError
			// Only use exit code if the program actually exited itself. With
			// this programs that are killed by be the kernel (e.g. the OOM
			// killer) result in errors, as they should.
			if errors.As(err, &exitErr) && exitErr.Exited() {
				return exitErr.ExitCode(), nil
			}

			return 0, fmt.Errorf("main program: %w", err)
		}

		return 0, nil
	})
}
