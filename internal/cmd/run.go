// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/signal"
	"syscall"

	"github.com/aibor/virtrun/internal/qemu"
	"github.com/aibor/virtrun/internal/virtrun"
)

func run(args []string, outWriter, errWriter io.Writer) error {
	flags := newFlags(args[0], errWriter)

	err := flags.ParseArgs(PrependEnvArgs(args[1:]))
	if err != nil {
		return fmt.Errorf("parse args: %w", err)
	}

	err = Validate(flags.spec)
	if err != nil {
		return fmt.Errorf("validate: %w", err)
	}

	setupLogging(errWriter, flags.Debug())

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGABRT,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGHUP,
	)
	defer cancel()

	err = virtrun.Run(ctx, flags.spec, outWriter, errWriter)
	if err != nil {
		return fmt.Errorf("run: %w", err)
	}

	return nil
}

func handleRunError(err error, errWriter io.Writer) int {
	if err == nil {
		return 0
	}

	// [ErrHelp] is returned when help is requested. So exit without error
	// in this case.
	if errors.Is(err, ErrHelp) {
		return 0
	}

	exitCode := -1

	// ParseArgs already prints errors, so we just exit without an error.
	if errors.Is(err, &ParseArgsError{}) {
		return exitCode
	}

	var qemuCmdErr *qemu.CommandError

	if errors.As(err, &qemuCmdErr) {
		if qemuCmdErr.ExitCode != 0 {
			exitCode = qemuCmdErr.ExitCode
		}
	}

	// Do not print the error in case the guest process ran successfully and
	// the guest properly communicated a non-zero exit code.
	if errors.Is(err, qemu.ErrGuestNonZeroExitCode) {
		return exitCode
	}

	fmt.Fprintf(errWriter, "Error [virtrun]: %v\n", err)

	return exitCode
}

func Run(args []string, outWriter, errWriter io.Writer) int {
	err := run(args, outWriter, errWriter)
	return handleRunError(err, errWriter)
}
