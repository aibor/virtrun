package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	// TODO: Replace with stdlib slices with go 1.21.
	"golang.org/x/exp/slices"

	"github.com/aibor/virtrun/internal/initramfs"
	"github.com/aibor/virtrun/internal/qemu"
	"github.com/aibor/virtrun/sysinit"
)

func run() (int, error) {
	var (
		cfg config
		err error
	)

	arch := os.Getenv("GOARCH")
	if arch == "" {
		arch = runtime.GOARCH
	}

	cfg.qemuCmd, err = qemu.CommandFor(arch)
	if err != nil {
		return 1, err
	}

	cfg.qemuCmd.Kernel = os.Getenv("QEMU_KERNEL")

	// ParseArgs already prints errors, so we just exit.
	if err := cfg.parseArgs(os.Args); err != nil {
		if err == flag.ErrHelp {
			return 0, nil
		}
		return 1, nil
	}

	for _, file := range cfg.binaries {
		if _, err := os.Stat(file); errors.Is(err, os.ErrNotExist) {
			return 1, fmt.Errorf("file %s doesn't exist.", file)
		}
	}

	if err := cfg.qemuCmd.Validate(); err != nil {
		return 1, fmt.Errorf("validate qemu command: %v", err)
	}
	if _, err := exec.LookPath(cfg.qemuCmd.Binary); errors.Is(err, os.ErrNotExist) {
		return 1, fmt.Errorf("kernel file %s doesn't exist.", cfg.qemuCmd.Kernel)
	}
	if _, err := os.Stat(cfg.qemuCmd.Kernel); errors.Is(err, os.ErrNotExist) {
		return 1, fmt.Errorf("qemu binary %s: %v", cfg.qemuCmd.Binary, err)
	}

	if !cfg.noGoTestFlagRewrite {
		qemu.ProcessGoTestFlags(&cfg.qemuCmd)
	}

	var archive *initramfs.Archive
	if cfg.standalone {
		archive = initramfs.New(cfg.binaries[0])
		cfg.binaries = slices.Delete(cfg.binaries, 0, 1)
	} else {
		if runtime.GOARCH != arch {
			return 1, fmt.Errorf(
				"using self as init only available with native architecture",
			)
		}
		var self string
		self, err = os.Executable()
		if err != nil {
			return 1, fmt.Errorf("get own path: %v", err)
		}
		archive = initramfs.New(self)
	}

	if err := archive.AddFiles(cfg.binaries...); err != nil {
		return 1, fmt.Errorf("add binares: %v", err)
	}

	if err := archive.AddRequiredSharedObjects(); err != nil {
		return 1, fmt.Errorf("add libs: %v", err)
	}

	archiveFile, err := os.CreateTemp("", "initramfs")
	if err != nil {
		return 1, fmt.Errorf("create initramfs archive file: %v", err)
	}

	if err := archive.WriteCPIO(archiveFile); err != nil {
		archiveFile.Close()
		_ = os.Remove(archiveFile.Name())
		return 1, fmt.Errorf("write initramfs archive: %v", err)
	}
	archiveFile.Close()

	cfg.qemuCmd.Initrd = archiveFile.Name()
	defer func() {
		_ = os.Remove(cfg.qemuCmd.Initrd)
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

	rc, err := cfg.qemuCmd.Run(ctx)
	if err != nil {
		return rc, fmt.Errorf("running QEMU command: %v", err)
	}

	return rc, nil
}

func runInit() (int, error) {
	err := sysinit.Run(func() (int, error) {
		dir := initramfs.FilesDir
		files, err := os.ReadDir(dir)
		if err != nil {
			return 98, err
		}

		paths := make([]string, len(files))
		for idx, f := range files {
			paths[idx] = filepath.Join(dir, f.Name())
		}

		return 0, sysinit.ExecParallel(paths, os.Args[1:], os.Stdout, os.Stderr)
	})
	if err == sysinit.ErrNotPidOne {
		return 127, err
	}
	return 126, err
}

func main() {
	f := run
	if os.Args[0] == "/init" {
		f = runInit
	}
	rc, err := f()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
	os.Exit(rc)
}
