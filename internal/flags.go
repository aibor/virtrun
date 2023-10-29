package internal

import (
	"flag"
	"fmt"
	"strconv"
)

// AddQEMUCommandFlags adds flags for configuring the QEMUCommand to the given
// FlagSet.
func AddQEMUCommandFlags(fs *flag.FlagSet, qemuCmd *QEMUCommand) {
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

	fs.BoolVar(
		&qemuCmd.Verbose,
		"verbose",
		qemuCmd.Verbose,
		"enable verbose guest system output",
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
}
