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

		if err := cmd.Run(); err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				return exitErr.ExitCode(), nil
			}

			return -1, fmt.Errorf("main: %w", err)
		}

		return 0, nil
	})
}
