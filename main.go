// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

// Package main provides the main virtrun entry point.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/aibor/virtrun/internal/cmd"
)

func run() int {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGABRT,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGHUP,
	)
	defer cancel()

	return cmd.Run(ctx, os.Args, cmd.IO{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
}

func main() {
	os.Exit(run())
}
