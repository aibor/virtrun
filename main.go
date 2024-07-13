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
	"syscall"

	"github.com/aibor/virtrun/internal/cmd"
)

//nolint:cyclop
func run() (int, error) {
	// Our init programs may return 127 and 126, so use 125 for indicating
	// issues.
	const errRC int = 125

	args, err := cmd.NewArgs(cmd.GetArch())
	if err != nil {
		return errRC, err
	}

	err = args.ParseArgs(
		os.Args[0],
		cmd.PrependEnvArgs(os.Args[1:]),
		os.Stderr,
	)
	if err != nil {
		// ParseArgs already prints errors, so we just exit without an error.
		if errors.Is(err, flag.ErrHelp) {
			return 0, nil
		}

		return errRC, nil
	}

	err = args.Validate()
	if err != nil {
		return errRC, err
	}

	// Build initramfs for the run.

	irfs, err := cmd.NewInitramfsArchive(args.InitramfsArgs)
	if err != nil {
		return errRC, fmt.Errorf("initramfs: %v", err)
	}

	defer func() {
		err := irfs.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: cleanup initramfs archive: %v", err)
		}
	}()

	cmd, err := cmd.NewQemuCommand(args.QemuArgs, irfs.Path)
	if err != nil {
		return errRC, err
	}

	if args.Debug {
		fmt.Fprintln(os.Stderr, "QEMU Args:")

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
