package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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
		if err == flag.ErrHelp {
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
		irfs = initramfs.New(cfg.binary)
	} else {
		// In the default wrapped mode a pre-compiled init is used that just
		// executes "/main".
		irfs, err = initramfs.NewWithInitFor(cfg.arch, cfg.binary)
		if err != nil {
			return errRC, fmt.Errorf("initramfs: %v", err)
		}
	}
	if err := irfs.AddFiles("data", cfg.files...); err != nil {
		return errRC, fmt.Errorf("add files: %v", err)
	}
	if err := irfs.AddRequiredSharedObjects(""); err != nil {
		return errRC, fmt.Errorf("add libs: %v", err)
	}

	if cfg.cmd.Initramfs, err = irfs.WriteToTempFile(""); err != nil {
		return errRC, fmt.Errorf("write initramfs: %v", err)
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
