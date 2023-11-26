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
	var (
		// Default to non 0 exit code, so 0 must be explicitly set for
		// successful execution.
		rc  int = 1
		err error
	)

	cfg, err := newConfig()
	if err != nil {
		return rc, err
	}

	// ParseArgs already prints errors, so we just exit.
	if err := cfg.parseArgs(os.Args); err != nil {
		if err == flag.ErrHelp {
			return 0, nil
		}
		return rc, nil
	}

	if err := cfg.validate(); err != nil {
		return rc, err
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
			return rc, fmt.Errorf("initramfs: %v", err)
		}
	}
	if err := irfs.AddFiles("data", cfg.files...); err != nil {
		return rc, fmt.Errorf("add files: %v", err)
	}
	if err := irfs.AddRequiredSharedObjects(""); err != nil {
		return rc, fmt.Errorf("add libs: %v", err)
	}

	if cfg.cmd.Initramfs, err = irfs.WriteToTempFile(""); err != nil {
		return rc, fmt.Errorf("write initramfs: %v", err)
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

	if rc, err = cfg.cmd.Run(ctx, os.Stdout, os.Stderr); err != nil {
		return rc, fmt.Errorf("run: %v", err)
	}

	return rc, nil
}

func main() {
	rc, err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
	os.Exit(rc)
}
