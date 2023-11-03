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

	fs.Func(
		"transport",
		fmt.Sprintf("io transport type: 0=isa, 1=pci, 2=mmio (default %d)", cmd.TransportType),
		func(s string) error {
			t, err := strconv.ParseUint(s, 10, 2)
			if err != nil {
				return err
			}
			if t > 2 {
				return fmt.Errorf("unknown transport type")
			}
			cmd.TransportType = TransportType(t)
			return nil
		},
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
			cmd.Memory = uint(mem)
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

			cmd.SMP = uint(mem)

			return nil
		},
	)
}
