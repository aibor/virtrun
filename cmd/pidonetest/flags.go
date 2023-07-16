package main

import (
	"flag"
	"fmt"
	"strconv"
	"strings"

	"github.com/aibor/go-pidonetest/internal"
)

func parseArgs(args []string, testBinaryPath *string, qemuCmd *internal.QEMUCommand) error {
	fsName := fmt.Sprintf("%s [flags...] [testbinary] [testflags...]", args[0])
	fs := flag.NewFlagSet(fsName, flag.ContinueOnError)

	fs.StringVar(
		&qemuCmd.Binary,
		"qemu-bin",
		qemuCmd.Binary,
		"QEMU binary to use",
	)

	fs.StringVar(
		&qemuCmd.Kernel,
		"kernel",
		qemuCmd.Kernel,
		"path to kernel to use",
	)

	fs.StringVar(
		&qemuCmd.Machine,
		"machine",
		qemuCmd.Machine,
		"QEMU machine type to use",
	)

	fs.StringVar(
		&qemuCmd.CPU,
		"cpu",
		qemuCmd.CPU,
		"QEMU cpu type to use",
	)

	fs.BoolVar(
		&qemuCmd.NoKVM,
		"nokvm",
		qemuCmd.NoKVM,
		"disable hardware support",
	)

	fs.Func(
		"memory",
		fmt.Sprintf("memory (in MB) for the QEMU VM (default %dMB)", qemuCmd.Memory),
		func(s string) error {
			mem, err := strconv.ParseUint(s, 10, 16)
			if err != nil {
				return err
			}
			if mem < 128 {
				return fmt.Errorf("less than 128 MB is not sufficient")
			}

			qemuCmd.Memory = uint16(mem)

			return nil
		},
	)

	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	posArgs := fs.Args()
	if len(posArgs) < 1 {
		fmt.Fprintln(fs.Output(), "no testbinary given")
		fs.Usage()
		return fmt.Errorf("no testbinary given")
	}

	*testBinaryPath = posArgs[0]

	// Catch coverage related paths and adjust them.
	for i := 1; i < len(posArgs); i++ {
		arg := posArgs[i]
		splits := strings.Split(arg, "=")
		switch splits[0] {
		case "-test.coverprofile":
			qemuCmd.SerialFiles = append(qemuCmd.SerialFiles, splits[1])
			splits[1] = "/dev/ttyS1"
			arg = strings.Join(splits, "=")
		case "-test.gocoverdir":
			splits[1] = "/tmp"
			arg = strings.Join(splits, "=")
		}
		qemuCmd.InitArgs = append(qemuCmd.InitArgs, arg)
	}

	return nil
}
