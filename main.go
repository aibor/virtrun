// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/aibor/virtrun/internal"
	"github.com/aibor/virtrun/internal/qemu"
)

// The init programs may return 127 and 126, so use 125 for indicating
// issues if the error does not return it's own return code.
const errExitCode = 125

func setupLogging(debug bool) {
	level := slog.LevelWarn
	if debug {
		level = slog.LevelDebug
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(
		os.Stderr,
		&slog.HandlerOptions{
			Level: level,
		},
	)))
}

func run() error {
	args, err := internal.NewArgs(internal.GetArch())
	if err != nil {
		return err
	}

	err = args.ParseArgs(
		os.Args[0],
		internal.PrependEnvArgs(os.Args[1:]),
		os.Stderr,
	)
	if err != nil {
		// ParseArgs already prints errors, so we just exit without an error.
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}

		return err
	}

	setupLogging(args.Debug)

	err = args.Validate()
	if err != nil {
		return err
	}

	// Build initramfs for the run.
	irfs, err := internal.NewInitramfsArchive(args.InitramfsArgs)
	if err != nil {
		return fmt.Errorf("initramfs: %w", err)
	}

	slog.Debug("Initramfs created", slog.String("path", irfs.Path))

	defer func() {
		err := irfs.Cleanup()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: cleanup initramfs archive: %v", err)
		}

		slog.Debug("Initramfs cleaned up", slog.String("path", irfs.Path))
	}()

	cmd, err := internal.NewQemuCommand(args.QemuArgs, irfs.Path)
	if err != nil {
		return err
	}

	slog.Debug("QEMU command",
		slog.String("qemu", cmd.Executable),
		slog.Any("args", cmd.Args()),
	)

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGABRT,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGHUP,
	)
	defer cancel()

	err = cmd.Run(ctx, os.Stdout, os.Stderr)
	if err != nil {
		return fmt.Errorf("run: %w", err)
	}

	return nil
}

func parseRC(err error) int {
	var qemuCmdErr *qemu.CommandError

	if errors.As(err, &qemuCmdErr) {
		return qemuCmdErr.ExitCode
	}

	return errExitCode
}

func main() {
	err := run()
	if err != nil {
		rc := parseRC(err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(rc)
	}
}
