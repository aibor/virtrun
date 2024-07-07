// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
)

const (
	memMin = 128
	memMax = 16384

	smpMin = 1
	smpMax = 16
)

func getArch() string {
	var arch string

	// Allow user to specify architecture by dedicated env var VIRTRUN_ARCH. It
	// can be empty, to suppress the GOARCH lookup and enforce the fallback to
	// the runtime architecture. If VIRTRUN_ARCH is not present, GOARCH will be
	// used. This is handy in case of cross-architecture go test invocations.
	for _, name := range []string{"VIRTRUN_ARCH", "GOARCH"} {
		if v, exists := os.LookupEnv(name); exists {
			arch = v

			break
		}
	}

	// Fallback to runtime architecture.
	if arch == "" {
		arch = runtime.GOARCH
	}

	return arch
}

// prependEnvArgs prepends virtrun arguments from the environment to the given
// list and returns the result. Because those args are prepended, the given
// args have precedence when parsed with [flag].
func prependEnvArgs(args []string) []string {
	envArgs := strings.Fields(os.Getenv("VIRTRUN_ARGS"))

	return append(envArgs, args...)
}

func run() (int, error) {
	// Our init programs may return 127 and 126, so use 125 for indicating
	// issues.
	const errRC int = 125

	args, err := newArgs(getArch())
	if err != nil {
		return errRC, err
	}

	err = args.parseArgs(os.Args[0], prependEnvArgs(os.Args[1:]), os.Stderr)
	if err != nil {
		// ParseArgs already prints errors, so we just exit without an error.
		if errors.Is(err, flag.ErrHelp) {
			return 0, nil
		}

		return errRC, nil
	}

	err = args.validate()
	if err != nil {
		return errRC, err
	}

	// Build initramfs for the run.
	irfs, err := newInitramfsArchive(args.initramfsArgs, args.arch)
	if err != nil {
		return errRC, fmt.Errorf("initramfs: %v", err)
	}
	defer irfs.Close()

	cmd, err := newCommand(args.qemuArgs, irfs.path)
	if err != nil {
		return errRC, err
	}

	if args.debug {
		fmt.Fprintln(os.Stdout, "QEMU Args:")

		for _, arg := range cmd.Args() {
			fmt.Fprintln(os.Stderr, arg)
		}
	}

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGABRT,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGHUP,
	)
	defer cancel()

	guestRC, err := cmd.Run(ctx, os.Stdout, os.Stderr)
	if err != nil {
		return errRC, fmt.Errorf("run: %v", err)
	}

	return guestRC, nil
}

func main() {
	rc, err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}

	os.Exit(rc)
}
