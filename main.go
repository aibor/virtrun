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

	"github.com/aibor/virtrun/internal/initprog"
	"github.com/aibor/virtrun/internal/initramfs"
)

func run() (int, error) {
	// Our init programs may return 127 and 126, so use 125 for indicating
	// issues.
	const errRC int = 125

	cfg, err := newConfig()
	if err != nil {
		return errRC, err
	}

	// ParseArgs already prints errors, so we just exit.
	if err := cfg.parseArgs(os.Args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0, nil
		}

		return errRC, nil
	}

	if err := cfg.validate(); err != nil {
		return errRC, err
	}

	// In order to be useful with "go test -exec", rewrite the file based flags
	// so the output can be passed from guest to kernel via consoles.
	if !cfg.noGoTestFlagRewrite {
		cfg.cmd.ProcessGoTestFlags()
	}

	// Build initramfs for the run.
	var irfs *initramfs.Initramfs
	if cfg.standalone {
		// In standalone mode, the first file (which might be the only one)
		// is supposed to work as an init matching our requirements.
		irfs = initramfs.New(initramfs.WithRealInitFile(cfg.binary))
	} else {
		// In the default wrapped mode a pre-compiled init is used that just
		// executes "/main".
		init, err := initprog.For(cfg.arch)
		if err != nil {
			return errRC, fmt.Errorf("embedded init: %v", err)
		}

		irfs = initramfs.New(initramfs.WithVirtualInitFile(init))

		if err := irfs.AddFile("/", "main", cfg.binary); err != nil {
			return errRC, fmt.Errorf("initramfs: add main file: %v", err)
		}
	}

	if err := irfs.AddFiles("data", cfg.files...); err != nil {
		return errRC, fmt.Errorf("initramfs: add files: %v", err)
	}

	if err := irfs.AddRequiredSharedObjects(); err != nil {
		return errRC, fmt.Errorf("initramfs: add libs: %v", err)
	}

	cfg.cmd.Initramfs, err = irfs.WriteToTempFile("")
	if err != nil {
		return errRC, fmt.Errorf("initramfs: write to temp file: %v", err)
	}

	defer func() {
		if cfg.keepInitramfs {
			fmt.Fprintf(os.Stderr, "initramfs kept at: %s\n", cfg.cmd.Initramfs)
		} else {
			_ = os.Remove(cfg.cmd.Initramfs)
		}
	}()

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGABRT,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGHUP,
	)
	defer cancel()

	guestRC, err := cfg.cmd.Run(ctx, os.Stdout, os.Stderr)
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
