package qemu

import (
	"flag"
	"fmt"
	"strconv"
)

// AddCommandFlags adds flags for configuring the [Command] to the given
// FlagSet.
func AddCommandFlags(fs *flag.FlagSet, cmd *Command) {
	fs.StringVar(
		&cmd.Binary,
		"qemu-bin",
		cmd.Binary,
		"QEMU binary to use",
	)

	fs.StringVar(
		&cmd.Kernel,
		"kernel",
		cmd.Kernel,
		"path to kernel to use",
	)

	fs.StringVar(
		&cmd.Machine,
		"machine",
		cmd.Machine,
		"QEMU machine type to use",
	)

	fs.StringVar(
		&cmd.CPU,
		"cpu",
		cmd.CPU,
		"QEMU cpu type to use",
	)

	fs.BoolVar(
		&cmd.NoKVM,
		"nokvm",
		cmd.NoKVM,
		"disable hardware support",
	)

	fs.BoolVar(
		&cmd.NoVirtioMMIO,
		"novmmio",
		cmd.NoVirtioMMIO,
		"use legacy isa pci devices instead of virtio-mmio",
	)

	fs.BoolVar(
		&cmd.Verbose,
		"verbose",
		cmd.Verbose,
		"enable verbose guest system output",
	)

	fs.Func(
		"memory",
		fmt.Sprintf("memory (in MB) for the QEMU VM (default %dMB)", cmd.Memory),
		func(s string) error {
			mem, err := strconv.ParseUint(s, 10, 16)
			if err != nil {
				return err
			}
			if mem < 128 {
				return fmt.Errorf("less than 128 MB is not sufficient")
			}

			cmd.Memory = uint16(mem)

			return nil
		},
	)

	fs.Func(
		"smp",
		fmt.Sprintf("number of CPUs for the QEMU VM (default %d)", cmd.SMP),
		func(s string) error {
			mem, err := strconv.ParseUint(s, 10, 4)
			if err != nil {
				return err
			}
			if mem < 1 {
				return fmt.Errorf("must not be less than 1")
			}

			cmd.SMP = uint8(mem)

			return nil
		},
	)
}
