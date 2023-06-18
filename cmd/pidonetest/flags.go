package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

func parseFlags(args []string, qemuCmd *QEMUCommand, testBinaryPath *string) bool {
	fs := flag.NewFlagSet(fmt.Sprintf("%s [flags...] [testbinary] [testflags...]", args[0]), flag.ContinueOnError)

	debug := fs.Bool(
		"debug",
		false,
		"enable debug output",
	)

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
		return false
	}

	if *debug {
		debugLog.SetOutput(os.Stderr)
	}

	posArgs := fs.Args()
	if len(posArgs) < 1 {
		fmt.Fprintln(fs.Output(), "no testbinary given")
		fs.Usage()
		return false
	}

	*testBinaryPath = posArgs[0]

	if len(posArgs) > 1 {
		qemuCmd.TestArgs = append(qemuCmd.TestArgs, posArgs[1:]...)
	}

	return true
}
