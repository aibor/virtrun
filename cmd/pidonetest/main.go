package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/aibor/go-pidonetest/internal"
)

func run() int {
	var testBinaryPath string
	qemuCmd := internal.QEMUCommand{
		Binary:  "qemu-system-x86_64",
		Kernel:  "/boot/vmlinuz-linux",
		Machine: "microvm",
		CPU:     "host",
		Memory:  128,
		NoKVM:   false,
	}

	// ParseArgs already prints errors, so we just exit.
	if err := parseArgs(os.Args, &testBinaryPath, &qemuCmd); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 1
	}

	if _, err := os.Stat(testBinaryPath); errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(os.Stderr, "testbinary file %s doesn't exist.\n", testBinaryPath)
		return 1
	}

	libs, err := internal.ResolveLinkedLibs(testBinaryPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving libs (Try again with CGO_ENABLED=0):\n%v\n", err)
		return 1
	}
	qemuCmd.Initrd, err = internal.CreateInitrd(testBinaryPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating intird (Try again with CGO_ENABLED=0):\n%v\n", err)
		return 1
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
		fmt.Fprintln(os.Stderr, "Error running QEMU command:\n", err)
	} else if err := qemuCmd.FixSerialFiles(); err != nil {
		fmt.Fprintln(os.Stderr, "Error fixing serial files:\n", err)
	}

	return rc
}

func main() {
	os.Exit(run())
}
