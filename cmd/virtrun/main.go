package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
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
		binaries   []string
		err        error
		standalone bool
	)

	arch := os.Getenv("GOARCH")
	if arch == "" {
		arch = runtime.GOARCH
	}

	qemuCmd, err := qemu.NewCommand(arch)
	if err != nil {
		return 1, err
	}

	qemuCmd.Kernel = os.Getenv("QEMU_KERNEL")

	// ParseArgs already prints errors, so we just exit.
	if err := parseArgs(os.Args, &binaries, qemuCmd, &standalone); err != nil {
		if err == flag.ErrHelp {
			return 0, nil
		}
		return 1, nil
	}

	if err := qemuCmd.Validate(); err != nil {
		return 1, fmt.Errorf("validate qemu command: %v", err)
	}

	for _, file := range binaries {
		if _, err := os.Stat(file); errors.Is(err, os.ErrNotExist) {
			return 1, fmt.Errorf("file %s doesn't exist.", file)
		}
	}

	if qemuCmd.Kernel == "" {
		msg := "no kernel specified (use env var QEMU_KERNEL or flag -kernel)"
		return 1, fmt.Errorf(msg)
	}
	if _, err := os.Stat(qemuCmd.Kernel); errors.Is(err, os.ErrNotExist) {
		return 1, fmt.Errorf("kernel file %s doesn't exist.", qemuCmd.Kernel)
	}

	var archive *initramfs.Archive
	if standalone {
		archive = initramfs.New(binaries[0])
		binaries = slices.Delete(binaries, 0, 1)
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

	if err := archive.AddFiles(binaries...); err != nil {
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

	qemuCmd.Initrd = archiveFile.Name()
	defer func() {
		_ = os.Remove(qemuCmd.Initrd)
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

	rc, err := qemuCmd.Run(ctx)
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
