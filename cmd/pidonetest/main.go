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
	"slices"
	"syscall"

	"github.com/aibor/initramfs"
	"github.com/aibor/pidonetest/internal"
	"github.com/aibor/pidonetest/sysinit"
)

func run() (int, error) {
	var (
		binaries   []string
		err        error
		standalone bool
	)

	qemuCmd, err := internal.NewQEMUCommand(runtime.GOARCH)
	if err != nil {
		return 1, err
	}

	qemuCmd.Kernel = os.Getenv("PIDONETEST_KERNEL")

	// ParseArgs already prints errors, so we just exit.
	if err := parseArgs(os.Args, &binaries, qemuCmd, &standalone); err != nil {
		if err == flag.ErrHelp {
			return 0, nil
		}
		return 1, nil
	}

	for _, file := range binaries {
		if _, err := os.Stat(file); errors.Is(err, os.ErrNotExist) {
			return 1, fmt.Errorf("file %s doesn't exist.", file)
		}
	}

	if qemuCmd.Kernel == "" {
		return 1, fmt.Errorf("no kernel specified (use env var PIDONETEST_KERNEL or flag -kernel)")
	}
	if _, err := os.Stat(qemuCmd.Kernel); errors.Is(err, os.ErrNotExist) {
		return 1, fmt.Errorf("kernel file %s doesn't exist.", qemuCmd.Kernel)
	}

	var archive *internal.Initramfs
	if standalone {
		archive = internal.NewInitramfs(binaries[0])
		binaries = slices.Delete(binaries, 0, 1)
	} else {
		var self string
		self, err = os.Executable()
		if err != nil {
			return 1, fmt.Errorf("get own path: %v", err)
		}
		archive = internal.NewInitramfs(self)
	}

	if err := archive.AddFiles(binaries...); err != nil {
		return 1, fmt.Errorf("add binares: %v", err)
	}

	qemuCmd.Initrd, err = archive.Write()
	if err != nil {
		return 1, fmt.Errorf("write initramfs: %v", err)
	}
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
	if !sysinit.IsPidOne() {
		return 127, sysinit.NotPidOneError
	}

	var err error
	defer sysinit.Poweroff(&err)

	err = sysinit.MountAll()
	if err != nil {
		return 126, err
	}

	files, err := os.ReadDir(initramfs.FilesDir)
	if err != nil {
		return 125, err
	}

	paths := make([]string, len(files))
	for idx, f := range files {
		paths[idx] = filepath.Join(initramfs.FilesDir, f.Name())
	}

	rc := 0
	err = sysinit.ExecParallel(paths, os.Args[1:], os.Stdout, os.Stderr)
	if err != nil {
		var eerr *exec.ExitError
		if errors.As(err, &eerr) {
			rc = eerr.ExitCode()
		} else {
			rc = 124
		}
		err = nil
	}
	sysinit.PrintRC(rc)

	return rc, nil
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
