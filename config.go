// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package main

import (
	"debug/elf"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/aibor/virtrun/internal/qemu"
)

type config struct {
	cmd                 *qemu.Command
	arch                string
	binary              string
	files               []string
	standalone          bool
	noGoTestFlagRewrite bool
	keepInitramfs       bool
}

func newConfig() (*config, error) {
	var arch string

	// Allow user to specify architecture by dedicated env var QEMU_ARCH. It
	// can be empty, to suppress the GOARCH lookup and enforce the fallback to
	// the runtime architecture. If QEMU_ARCH is not present, GOARCH will be
	// used. This is handy in case of cross-architecture go test invocations.
	for _, name := range []string{"QEMU_ARCH", "GOARCH"} {
		if v, exists := os.LookupEnv(name); exists {
			arch = v
			break
		}
	}
	// Fallback to runtime architecture.
	if arch == "" {
		arch = runtime.GOARCH
	}

	// Provision defaults for the requested architecture.
	cmd, err := qemu.NewCommand(arch)
	if err != nil {
		return nil, err
	}

	// Preset kernel from environment. Must be a kernel with the same
	// architecture QEMU is supposed to run and the binary that is given.
	cmd.Kernel = os.Getenv("QEMU_KERNEL")

	cfg := &config{
		arch: arch,
		cmd:  cmd,
	}

	return cfg, nil
}

func (cfg *config) parseArgs(args []string) error {
	fsName := fmt.Sprintf("%s [flags...] binary [initargs...]", args[0])
	fs := flag.NewFlagSet(fsName, flag.ContinueOnError)

	fs.StringVar(
		&cfg.cmd.Executable,
		"qemu-bin",
		cfg.cmd.Executable,
		"QEMU binary to use",
	)

	fs.StringVar(
		&cfg.cmd.Kernel,
		"kernel",
		cfg.cmd.Kernel,
		"path to kernel to use",
	)

	fs.StringVar(
		&cfg.cmd.Machine,
		"machine",
		cfg.cmd.Machine,
		"QEMU machine type to use",
	)

	fs.StringVar(
		&cfg.cmd.CPU,
		"cpu",
		cfg.cmd.CPU,
		"QEMU CPU type to use",
	)

	fs.BoolVar(
		&cfg.cmd.NoKVM,
		"nokvm",
		cfg.cmd.NoKVM,
		"disable hardware support",
	)

	fs.Func(
		"transport",
		fmt.Sprintf("io transport type: 0=isa, 1=pci, 2=mmio (default %d)", cfg.cmd.TransportType),
		func(s string) error {
			t, err := strconv.ParseUint(s, 10, 2)
			if err != nil {
				return err
			}
			if t > 2 {
				return fmt.Errorf("unknown transport type")
			}
			cfg.cmd.TransportType = qemu.TransportType(t)
			return nil
		},
	)

	fs.BoolVar(
		&cfg.cmd.Verbose,
		"verbose",
		cfg.cmd.Verbose,
		"enable verbose guest system output",
	)

	fs.Func(
		"memory",
		fmt.Sprintf("memory (in MB) for the QEMU VM (default %dMB)", cfg.cmd.Memory),
		func(s string) error {
			mem, err := strconv.ParseUint(s, 10, 16)
			if err != nil {
				return err
			}
			if mem < 128 {
				return fmt.Errorf("less than 128 MB is not sufficient")
			}
			cfg.cmd.Memory = uint(mem)
			return nil
		},
	)

	fs.Func(
		"smp",
		fmt.Sprintf("number of CPUs for the QEMU VM (default %d)", cfg.cmd.SMP),
		func(s string) error {
			mem, err := strconv.ParseUint(s, 10, 4)
			if err != nil {
				return err
			}
			if mem < 1 {
				return fmt.Errorf("must not be less than 1")
			}
			cfg.cmd.SMP = uint(mem)
			return nil
		},
	)

	fs.BoolVar(
		&cfg.standalone,
		"standalone",
		cfg.standalone,
		"run first given file as init itself. Use this if it has virtrun support built in.",
	)

	fs.BoolVar(
		&cfg.noGoTestFlagRewrite,
		"noGoTestFlagRewrite",
		cfg.noGoTestFlagRewrite,
		"disable automatic go test flag rewrite for file based output.",
	)

	fs.BoolVar(
		&cfg.keepInitramfs,
		"keepInitramfs",
		cfg.keepInitramfs,
		"do not delete initramfs once qemu is done. Intended for debugging. "+
			"The path to the file is printed on stderr",
	)

	fs.Func(
		"addFile",
		"file to add to guest's /data dir. Flag may be used more than once.",
		func(s string) error {
			if s == "" {
				return fmt.Errorf("file path must not be empty")
			}
			path, err := filepath.Abs(s)
			if err != nil {
				return err
			}
			cfg.files = append(cfg.files, path)
			return nil
		},
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

	if cfg.cmd.Kernel == "" {
		return failf("no kernel given (use env var QEMU_KERNEL or flag -kernel)")
	}

	// First positional argument is supposed to be a binary file.
	if len(fs.Args()) < 1 {
		return failf("no binary given")
	}
	var err error
	cfg.binary, err = filepath.Abs(fs.Args()[0])
	if err != nil {
		return failf("absolute path for %s: %v", fs.Args()[0], err)
	}

	// All further positional arguments after the binary file will be passed to
	// the guest system's init program.
	cfg.cmd.InitArgs = fs.Args()[1:]

	return nil
}

func (cfg *config) validate() error {
	// Do some simple input validation to catch most obvious issues.
	if err := cfg.cmd.Validate(); err != nil {
		return fmt.Errorf("validate qemu command: %v", err)
	}

	// Check files are actually present.
	if _, err := exec.LookPath(cfg.cmd.Executable); err != nil {
		return fmt.Errorf("check qemu binary: %v", err)
	}
	if _, err := os.Stat(cfg.cmd.Kernel); err != nil {
		return fmt.Errorf("check kernel file: %v", err)
	}
	for _, file := range cfg.files {
		if _, err := os.Stat(file); err != nil {
			return fmt.Errorf("check file: %v", err)
		}
	}

	// Do some deeper validation for the main binary.
	elfFile, err := elf.Open(cfg.binary)
	if err != nil {
		return fmt.Errorf("check main binary: %v", err)
	}
	defer elfFile.Close()

	if err := validateELF(elfFile.FileHeader, cfg.arch); err != nil {
		return fmt.Errorf("check main binary: %v", err)
	}

	return nil
}

// validateELF validates that ELF attributes match the requested architecture.
func validateELF(hdr elf.FileHeader, arch string) error {
	switch hdr.OSABI {
	case elf.ELFOSABI_NONE, elf.ELFOSABI_LINUX:
		// supported, pass
	default:
		return fmt.Errorf("OSABI not supported: %s", hdr.OSABI)
	}

	var archReq string
	switch hdr.Machine {
	case elf.EM_X86_64:
		archReq = "amd64"
	case elf.EM_AARCH64:
		archReq = "arm64"
	default:
		return fmt.Errorf("machine type not supported: %s", hdr.Machine)
	}
	if archReq != arch {
		return fmt.Errorf("machine %s not supported for %s", hdr.Machine, arch)
	}

	return nil
}
