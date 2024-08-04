// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package internal

import (
	"debug/elf"
	"errors"
	"flag"
	"fmt"
	"io"
	"os/exec"

	"github.com/aibor/virtrun/internal/qemu"
)

const (
	cpuDefault = "max"

	memMin     = 128
	memMax     = 16384
	memDefault = 256

	smpMin     = 1
	smpMax     = 16
	smpDefault = 1
)

// Set on build.
var (
	version = "dev"
	commit  = "none"    //nolint:gochecknoglobals
	date    = "unknown" //nolint:gochecknoglobals
)

type Config struct {
	Qemu      QemuConfig
	Initramfs InitramfsConfig
	Version   bool
	Debug     bool
}

func NewConfig(arch Arch) (Config, error) {
	var (
		qemuExecutable    string
		qemuMachine       string
		qemuTransportType qemu.TransportType
	)

	switch arch {
	case ArchAMD64:
		qemuExecutable = "qemu-system-x86_64"
		qemuMachine = "q35"
		qemuTransportType = qemu.TransportTypePCI
	case ArchARM64:
		qemuExecutable = "qemu-system-aarch64"
		qemuMachine = "virt"
		qemuTransportType = qemu.TransportTypeMMIO
	default:
		return Config{}, fmt.Errorf("arch [%s]: %w", arch, errors.ErrUnsupported)
	}

	args := Config{
		Qemu: QemuConfig{
			Executable:    qemuExecutable,
			Machine:       qemuMachine,
			TransportType: qemuTransportType,
			CPU:           cpuDefault,
			Memory: LimitedUintFlag{
				memDefault,
				memMin,
				memMax,
				"MB",
			},
			SMP: LimitedUintFlag{
				smpDefault,
				smpMin,
				smpMax,
				"",
			},
			NoKVM: !arch.KVMAvailable(),
			ExtraArgs: []qemu.Argument{
				qemu.UniqueArg("display", "none"),
				qemu.UniqueArg("monitor", "none"),
				qemu.UniqueArg("no-reboot", ""),
				qemu.UniqueArg("nodefaults", ""),
				qemu.UniqueArg("no-user-config", ""),
			},
		},
		Initramfs: InitramfsConfig{
			Arch: arch,
		},
	}

	return args, nil
}

func (c *Config) newFlagset(self string) *flag.FlagSet {
	fsName := self + " [flags...] binary [initargs...]"
	fs := flag.NewFlagSet(fsName, flag.ContinueOnError)

	fs.StringVar(
		&c.Qemu.Executable,
		"qemu-bin",
		c.Qemu.Executable,
		"QEMU binary to use",
	)

	fs.TextVar(
		&c.Qemu.Kernel,
		"kernel",
		c.Qemu.Kernel,
		"path to kernel to use",
	)

	fs.StringVar(
		&c.Qemu.Machine,
		"machine",
		c.Qemu.Machine,
		"QEMU machine type to use",
	)

	fs.StringVar(
		&c.Qemu.CPU,
		"cpu",
		c.Qemu.CPU,
		"QEMU CPU type to use",
	)

	fs.BoolVar(
		&c.Qemu.NoKVM,
		"nokvm",
		c.Qemu.NoKVM,
		"disable hardware support",
	)

	fs.TextVar(
		&c.Qemu.TransportType,
		"transport",
		c.Qemu.TransportType,
		"io transport type: isa, pci, mmio",
	)

	fs.BoolVar(
		&c.Qemu.Verbose,
		"verbose",
		c.Qemu.Verbose,
		"enable verbose guest system output",
	)

	fs.TextVar(
		&c.Qemu.Memory,
		"memory",
		c.Qemu.Memory,
		"memory (in MB) for the QEMU VM",
	)

	fs.TextVar(
		&c.Qemu.SMP,
		"smp",
		c.Qemu.SMP,
		"number of CPUs for the QEMU VM",
	)

	fs.BoolVar(
		&c.Initramfs.StandaloneInit,
		"standalone",
		c.Initramfs.StandaloneInit,
		"run first given file as init itself. Use this if it has virtrun support built in.",
	)

	fs.BoolVar(
		&c.Qemu.NoGoTestFlagRewrite,
		"noGoTestFlagRewrite",
		c.Qemu.NoGoTestFlagRewrite,
		"disable automatic go test flag rewrite for file based output.",
	)

	fs.BoolVar(
		&c.Initramfs.Keep,
		"keepInitramfs",
		c.Initramfs.Keep,
		"do not delete initramfs once qemu is done. Intended for debugging. "+
			"The path to the file is printed on stderr",
	)

	fs.Var(
		&c.Initramfs.Files,
		"addFile",
		"file to add to guest's /data dir. Flag may be used more than once.",
	)

	fs.Var(
		&c.Initramfs.Modules,
		"addModule",
		"kernel module to add to guest. Flag may be used more than once.",
	)

	fs.BoolVar(
		&c.Version,
		"version",
		c.Version,
		"show version and exit",
	)

	fs.BoolVar(
		&c.Debug,
		"debug",
		c.Debug,
		"enable debug output",
	)

	return fs
}

func (c *Config) ParseArgs(name string, args []string, output io.Writer) error {
	fs := c.newFlagset(name)
	fs.SetOutput(output)

	// Parses arguments up to the first one that is not prefixed with a "-" or
	// is "--".
	if err := fs.Parse(args); err != nil {
		return &ParseArgsError{msg: "flag parse: %w", err: err}
	}

	printf := func(format string, a ...any) string {
		msg := fmt.Sprintf(format, a...)
		fmt.Fprintln(fs.Output(), msg)

		return msg
	}

	// Fail like flag does.
	failf := func(format string, a ...any) error {
		msg := printf(format, a...)

		fs.Usage()

		return &ParseArgsError{msg: msg}
	}

	// With version flag, just print the version and exit. Using [flag.ErrHelp]
	// the main binary is supposed to return with a non error exit code.
	if c.Version {
		msgFmt := "virtrun %s\n  commit %s\n  built at %s"
		printf(msgFmt, version, commit, date)

		return &ParseArgsError{err: flag.ErrHelp}
	}

	if c.Qemu.Kernel == "" {
		return failf("no kernel given (use -kernel)")
	}

	// First positional argument is supposed to be a binary file.
	if len(fs.Args()) < 1 {
		return failf("no binary given")
	}

	binary, err := AbsoluteFilePath(fs.Args()[0])
	if err != nil {
		return failf("binary path: %w", err)
	}

	c.Initramfs.Binary = binary

	// All further positional arguments after the binary file will be passed to
	// the guest system's init program.
	c.Qemu.InitArgs = fs.Args()[1:]

	return nil
}

func (c *Config) Validate() error {
	// Check files are actually present.
	if _, err := exec.LookPath(c.Qemu.Executable); err != nil {
		return fmt.Errorf("check qemu binary: %w", err)
	}

	if err := c.Qemu.Kernel.check(); err != nil {
		return fmt.Errorf("check kernel file: %w", err)
	}

	for _, file := range c.Initramfs.Files {
		if err := FilePath(file).check(); err != nil {
			return fmt.Errorf("check file: %w", err)
		}
	}

	for _, file := range c.Initramfs.Modules {
		if err := FilePath(file).check(); err != nil {
			return fmt.Errorf("check module: %w", err)
		}
	}

	// Do some deeper validation for the main binary.
	elfFile, err := elf.Open(string(c.Initramfs.Binary))
	if err != nil {
		return fmt.Errorf("check main binary: %w", err)
	}
	defer elfFile.Close()

	if err := validateELF(elfFile.FileHeader, c.Initramfs.Arch); err != nil {
		return fmt.Errorf("check main binary: %w", err)
	}

	return nil
}

// validateELF validates that ELF attributes match the requested architecture.
func validateELF(hdr elf.FileHeader, arch Arch) error {
	switch hdr.OSABI {
	case elf.ELFOSABI_NONE, elf.ELFOSABI_LINUX:
		// supported, pass
	default:
		return fmt.Errorf("OSABI not supported: %s", hdr.OSABI)
	}

	var archReq Arch

	switch hdr.Machine {
	case elf.EM_X86_64:
		archReq = ArchAMD64
	case elf.EM_AARCH64:
		archReq = ArchARM64
	default:
		return fmt.Errorf("machine type not supported: %s", hdr.Machine)
	}

	if archReq != arch {
		return fmt.Errorf("machine %s not supported for %s", hdr.Machine, arch)
	}

	return nil
}
