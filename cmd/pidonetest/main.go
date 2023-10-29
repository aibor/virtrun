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

	"github.com/aibor/initramfs"
	"github.com/aibor/pidonetest/internal"
	"github.com/aibor/pidonetest/sysinit"
)

func run() (int, error) {
	var (
		testBinaryPath string
		err            error
		standalone     bool
	)

	qemuCmd, err := internal.NewQEMUCommand(runtime.GOARCH)
	if err != nil {
		return 1, err
	}

	qemuCmd.Kernel = os.Getenv("PIDONETEST_KERNEL")

	// ParseArgs already prints errors, so we just exit.
	if err := parseArgs(os.Args, &testBinaryPath, qemuCmd, &standalone); err != nil {
		if err == flag.ErrHelp {
			return 0, nil
		}
		return 1, nil
	}

	if _, err := os.Stat(testBinaryPath); errors.Is(err, os.ErrNotExist) {
		return 1, fmt.Errorf("testbinary file %s doesn't exist.", testBinaryPath)
	}

	if qemuCmd.Kernel == "" {
		return 1, fmt.Errorf("no kernel specified (use env var PIDONETEST_KERNEL or flag -kernel)")
	}
	if _, err := os.Stat(qemuCmd.Kernel); errors.Is(err, os.ErrNotExist) {
		return 1, fmt.Errorf("kernel file %s doesn't exist.", qemuCmd.Kernel)
	}

	var archive *internal.Initramfs
	if standalone {
		archive = internal.NewInitramfs(testBinaryPath)
	} else {
		var self string
		self, err = os.Executable()
		if err != nil {
			return 1, fmt.Errorf("get own path: %v", err)
		}
		archive = internal.NewInitramfs(self)
		if err := archive.AddFile("test", testBinaryPath); err != nil {
			return 1, fmt.Errorf("add test binary: %v", err)
		}
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

	if err := qemuCmd.FixSerialFiles(); err != nil {
		return rc, fmt.Errorf(" fixing serial files: %v", err)
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

	rc := 0
	path := filepath.Join(initramfs.FilesDir, "test")
	err = sysinit.Exec(path, os.Args[1:], os.Stdout, os.Stderr)
	if err != nil {
		if !errors.Is(err, &exec.ExitError{}) {
			return 125, err
		}
		rc = 1
	}
	sysinit.PrintRC(rc)

	return 0, nil
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
