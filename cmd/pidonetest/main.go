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
	"syscall"

	"github.com/aibor/go-pidonetest"
	"github.com/aibor/go-pidonetest/internal"
)

func run() (int, error) {
	var (
		testBinaryPath string
		err            error
		qemuCmd        = internal.QEMUCommand{
			Binary:  "qemu-system-x86_64",
			Kernel:  "/boot/vmlinuz-linux",
			Machine: "microvm",
			CPU:     "host",
			Memory:  256,
			NoKVM:   false,
		}
		wrap bool
	)

	// ParseArgs already prints errors, so we just exit.
	if err := parseArgs(os.Args, &testBinaryPath, &qemuCmd, &wrap); err != nil {
		if err == flag.ErrHelp {
			return 0, nil
		}
		return 1, nil
	}

	if _, err := os.Stat(testBinaryPath); errors.Is(err, os.ErrNotExist) {
		return 1, fmt.Errorf("testbinary file %s doesn't exist.", testBinaryPath)
	}

	if wrap {
		var self string
		self, err = os.Executable()
		if err != nil {
			return 1, fmt.Errorf("get own path: %v", err)
		}
		qemuCmd.Initrd, err = internal.CreateInitrd(self, testBinaryPath)
	} else {
		qemuCmd.Initrd, err = internal.CreateInitrd(testBinaryPath)
	}
	if err != nil {
		return 1, fmt.Errorf("creating intird (Try again with CGO_ENABLED=0): %v", err)
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
	if !pidonetest.IsPidOne() {
		return 127, pidonetest.NotPidOneError
	}
	defer pidonetest.Poweroff()

	if err := pidonetest.MountAll(); err != nil {
		return 126, fmt.Errorf("mounting file systems: %v", err)
	}

	files, err := os.ReadDir("files")
	if err != nil {
		return 126, fmt.Errorf("read test files: %v", err)
	}

	rc := 0
	for _, f := range files {
		path := filepath.Join("files", f.Name())
		cmd := exec.Command(path, os.Args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			if !errors.Is(err, &exec.ExitError{}) {
				return 125, fmt.Errorf("running %s: %v", path, err)
			}
			rc = 1
		}
	}
	pidonetest.PrintPidOneTestRC(rc)

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
