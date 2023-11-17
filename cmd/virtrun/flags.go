package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aibor/virtrun/internal/qemu"
)

type config struct {
	binaries            []string
	qemuCmd             qemu.Command
	standalone          bool
	noGoTestFlagRewrite bool
}

func (cfg *config) parseArgs(args []string) error {
	fsName := fmt.Sprintf("%s [flags...] binaries... [initflags...]", args[0])
	fs := flag.NewFlagSet(fsName, flag.ContinueOnError)

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
		"QEMU CPU type to use",
	)

	fs.BoolVar(
		&cfg.qemuCmd.NoKVM,
		"nokvm",
		cfg.qemuCmd.NoKVM,
		"disable hardware support",
	)

	fs.Func(
		"transport",
		fmt.Sprintf("io transport type: 0=isa, 1=pci, 2=mmio (default %d)", cfg.qemuCmd.TransportType),
		func(s string) error {
			t, err := strconv.ParseUint(s, 10, 2)
			if err != nil {
				return err
			}
			if t > 2 {
				return fmt.Errorf("unknown transport type")
			}
			cfg.qemuCmd.TransportType = qemu.TransportType(t)
			return nil
		},
	)

	fs.BoolVar(
		&cfg.qemuCmd.Verbose,
		"verbose",
		cfg.qemuCmd.Verbose,
		"enable verbose guest system output",
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
			cfg.qemuCmd.Memory = uint(mem)
			return nil
		},
	)

	fs.Func(
		"smp",
		fmt.Sprintf("number of CPUs for the QEMU VM (default %d)", cfg.qemuCmd.SMP),
		func(s string) error {
			mem, err := strconv.ParseUint(s, 10, 4)
			if err != nil {
				return err
			}
			if mem < 1 {
				return fmt.Errorf("must not be less than 1")
			}

			cfg.qemuCmd.SMP = uint(mem)

			return nil
		},
	)

	fs.BoolVar(
		&cfg.standalone,
		"standalone",
		cfg.standalone,
		"run first given binary as init itself. Use this if it has virtrun support built in.",
	)

	fs.BoolVar(
		&cfg.noGoTestFlagRewrite,
		"noGoTestFlagRewrite",
		cfg.noGoTestFlagRewrite,
		"disable automatic go test flag rewrite for file based output.",
	)

	// Parses arguments up to the first one that is not prefixed with a "-" or
	// is "--".
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	// Fail like flag does.
	failf := func(format string, a ...any) error {
		msg := fmt.Sprintf(format, a...)
		fmt.Fprintln(fs.Output(), msg)
		fs.Usage()
		return fmt.Errorf(msg)
	}

	if cfg.qemuCmd.Kernel == "" {
		return failf("no kernel given (use env var QEMU_KERNEL or flag -kernel)")
	}

	// Consider all positional arguments until one begins with "-" as binary
	// files that should be added to the initramfs. All further arguments
	// are added as [qemu.Command.InitArgs] that will be passed to the guest
	// system's init program.
	var binariesDone bool
	for _, arg := range fs.Args() {
		switch {
		case strings.HasPrefix(arg, "-"):
			binariesDone = true
			fallthrough
		case binariesDone:
			cfg.qemuCmd.InitArgs = append(cfg.qemuCmd.InitArgs, arg)
		default:
			path, err := filepath.Abs(arg)
			if err != nil {
				return failf("absolute path for %s: %v", arg, err)
			}
			cfg.binaries = append(cfg.binaries, path)
		}
	}

	if len(cfg.binaries) < 1 {
		return failf("no binary given")
	}

	return nil
}
