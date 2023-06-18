package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

func parseFlags(args []string, cfg *config) bool {
	fs := flag.NewFlagSet(fmt.Sprintf("%s [flags...] [testbinary] [testflags...]", args[0]), flag.ContinueOnError)

	debug := fs.Bool(
		"debug",
		false,
		"enable debug output",
	)

	fs.StringVar(
		&cfg.qemuCmd.Binary,
		"qemu-bin",
		cfg.qemuCmd.Binary,
		"QEMU binary to use",
	)

	fs.StringVar(
		&cfg.qemuCmd.Kernel,
		"kernel",
		cfg.qemuCmd.Kernel,
		"path to kernel to use",
	)

	fs.StringVar(
		&cfg.qemuCmd.Machine,
		"machine",
		cfg.qemuCmd.Machine,
		"QEMU machine type to use",
	)

	fs.StringVar(
		&cfg.qemuCmd.CPU,
		"cpu",
		cfg.qemuCmd.CPU,
		"QEMU cpu type to use",
	)

	fs.BoolVar(
		&cfg.qemuCmd.NoKVM,
		"nokvm",
		cfg.qemuCmd.NoKVM,
		"disable hardware support",
	)

	fs.Func(
		"memory",
		fmt.Sprintf("memory (in MB) for the QEMU VM (default %dMB)", cfg.qemuCmd.Memory),
		func(s string) error {
			mem, err := strconv.ParseUint(s, 10, 16)
			if err != nil {
				return err
			}
			if mem < 128 {
				return fmt.Errorf("less than 128 MB is not sufficient")
			}

			cfg.qemuCmd.Memory = uint16(mem)

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

	cfg.testBinaryPath = posArgs[0]

	if len(posArgs) > 1 {
		cfg.qemuCmd.TestArgs = append(cfg.qemuCmd.TestArgs, posArgs[1:]...)
	}

	return true
}
