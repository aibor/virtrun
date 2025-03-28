// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/aibor/virtrun/internal/qemu"
	"github.com/aibor/virtrun/internal/virtrun"
)

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	flags := newFlags(args[0], stderr)

	err := flags.ParseArgs(PrependEnvArgs(args[1:]))
	if err != nil {
		return fmt.Errorf("parse args: %w", err)
	}

	err = Validate(flags.spec)
	if err != nil {
		return fmt.Errorf("validate: %w", err)
	}

	setupLogging(stderr, flags.Debug())

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGABRT,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGHUP,
	)
	defer cancel()

	err = virtrun.Run(ctx, flags.spec, stdin, stdout, stderr)
	if err != nil {
		return fmt.Errorf("run: %w", err)
	}

	return nil
}

func handleRunError(err error) int {
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

	slog.Error(err.Error())

	return exitCode
}

func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	err := run(args, stdin, stdout, stderr)
	return handleRunError(err)
}
