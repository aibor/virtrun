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
	"strconv"

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

type limitedUintFlag struct {
	Value    uint
	min, max uint64
	unit     string
}

func (u limitedUintFlag) MarshalText() ([]byte, error) {
	return []byte(strconv.Itoa(int(u.Value)) + u.unit), nil
}

var ErrValueOutsideRange = errors.New("value is outside of range")

func (u *limitedUintFlag) UnmarshalText(text []byte) error {
	value, err := strconv.ParseUint(string(text), 10, 0)
	if err != nil {
		return err
	}

	if u.min > 0 && value < u.min {
		return fmt.Errorf("%d < %d: %w", value, u.min, ErrValueOutsideRange)
	}

	if u.max > 0 && value > u.max {
		return fmt.Errorf("%d > %d: %w", value, u.max, ErrValueOutsideRange)
	}

	u.Value = uint(value)

	return nil
}

type transportType struct {
	qemu.TransportType
}

var errInvalidTransportType = errors.New("unknown transport type")

// MarshalText implements [encoding.TextMarshaler].
func (t transportType) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnarshalText implements [encoding.TextUnmarshaler].
func (t *transportType) UnmarshalText(text []byte) error {
	s := string(text)

	types := []qemu.TransportType{
		qemu.TransportTypeISA,
		qemu.TransportTypePCI,
		qemu.TransportTypeMMIO,
	}
	for _, qt := range types {
		if s == strconv.Itoa(int(qt)) || s == qt.String() {
			t.TransportType = qt

			return nil
		}
	}

	return errInvalidTransportType
}

type Args struct {
	QemuArgs
	InitramfsArgs
	Version bool
	Debug   bool
}

func NewArgs(arch string) (Args, error) {
	var (
		qemuBin   string
		machine   string
		transport qemu.TransportType
	)

	switch arch {
	case "amd64":
		qemuBin = "qemu-system-x86_64"
		machine = "q35"
		transport = qemu.TransportTypePCI
	case "arm64":
		qemuBin = "qemu-system-aarch64"
		machine = "virt"
		transport = qemu.TransportTypeMMIO
	default:
		return Args{}, fmt.Errorf("arch [%s]: %w", arch, errors.ErrUnsupported)
	}

	args := Args{
		QemuArgs: QemuArgs{
			QemuBin:   qemuBin,
			Machine:   machine,
			Transport: transportType{transport},
			CPU:       cpuDefault,
			Memory: limitedUintFlag{
				memDefault,
				memMin,
				memMax,
				"MB",
			},
			SMP: limitedUintFlag{
				smpDefault,
				smpMin,
				smpMax,
				"",
			},
			NoKVM: !qemu.KVMAvailableFor(arch),
			ExtraArgs: []qemu.Argument{
				qemu.UniqueArg("display", "none"),
				qemu.UniqueArg("monitor", "none"),
				qemu.UniqueArg("no-reboot", ""),
				qemu.UniqueArg("nodefaults", ""),
				qemu.UniqueArg("no-user-config", ""),
			},
		},
		InitramfsArgs: InitramfsArgs{
			Arch: arch,
		},
	}

	return args, nil
}

func (a *Args) newFlagset(self string) *flag.FlagSet {
	fsName := self + " [flags...] binary [initargs...]"
	fs := flag.NewFlagSet(fsName, flag.ContinueOnError)

	fs.StringVar(
		&a.QemuBin,
		"qemu-bin",
		a.QemuBin,
		"QEMU binary to use",
	)

	fs.TextVar(
		&a.Kernel,
		"kernel",
		a.Kernel,
		"path to kernel to use",
	)

	fs.StringVar(
		&a.Machine,
		"machine",
		a.Machine,
		"QEMU machine type to use",
	)

	fs.StringVar(
		&a.CPU,
		"cpu",
		a.CPU,
		"QEMU CPU type to use",
	)

	fs.BoolVar(
		&a.NoKVM,
		"nokvm",
		a.NoKVM,
		"disable hardware support",
	)

	fs.TextVar(
		&a.Transport,
		"transport",
		a.Transport,
		"io transport type: isa/0, pci/1, mmio/2",
	)

	fs.BoolVar(
		&a.Verbose,
		"verbose",
		a.Verbose,
		"enable verbose guest system output",
	)

	fs.TextVar(
		&a.Memory,
		"memory",
		a.Memory,
		"memory (in MB) for the QEMU VM",
	)

	fs.TextVar(
		&a.SMP,
		"smp",
		a.SMP,
		"number of CPUs for the QEMU VM",
	)

	fs.BoolVar(
		&a.Standalone,
		"standalone",
		a.Standalone,
		"run first given file as init itself. Use this if it has virtrun support built in.",
	)

	fs.BoolVar(
		&a.NoGoTestFlagRewrite,
		"noGoTestFlagRewrite",
		a.NoGoTestFlagRewrite,
		"disable automatic go test flag rewrite for file based output.",
	)

	fs.BoolVar(
		&a.KeepInitramfs,
		"keepInitramfs",
		a.KeepInitramfs,
		"do not delete initramfs once qemu is done. Intended for debugging. "+
			"The path to the file is printed on stderr",
	)

	fs.Var(
		&a.Files,
		"addFile",
		"file to add to guest's /data dir. Flag may be used more than once.",
	)

	fs.Var(
		&a.Modules,
		"addModule",
		"kernel module to add to guest. Flag may be used more than once.",
	)

	fs.BoolVar(
		&a.Version,
		"version",
		a.Version,
		"show version and exit",
	)

	fs.BoolVar(
		&a.Debug,
		"debug",
		a.Debug,
		"enable debug output",
	)

	return fs
}

func (a *Args) ParseArgs(name string, args []string, output io.Writer) error {
	fs := a.newFlagset(name)
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
	if a.Version {
		msgFmt := "virtrun %s\n  commit %s\n  built at %s"
		printf(msgFmt, version, commit, date)

		return &ParseArgsError{err: flag.ErrHelp}
	}

	if a.Kernel == "" {
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

	a.Binary = binary

	// All further positional arguments after the binary file will be passed to
	// the guest system's init program.
	a.InitArgs = fs.Args()[1:]

	return nil
}

func (a *Args) Validate() error {
	// Check files are actually present.
	if _, err := exec.LookPath(a.QemuBin); err != nil {
		return fmt.Errorf("check qemu binary: %w", err)
	}

	if err := a.Kernel.check(); err != nil {
		return fmt.Errorf("check kernel file: %w", err)
	}

	for _, file := range a.Files {
		if err := FilePath(file).check(); err != nil {
			return fmt.Errorf("check file: %w", err)
		}
	}

	for _, file := range a.Modules {
		if err := FilePath(file).check(); err != nil {
			return fmt.Errorf("check module: %w", err)
		}
	}

	// Do some deeper validation for the main binary.
	elfFile, err := elf.Open(string(a.Binary))
	if err != nil {
		return fmt.Errorf("check main binary: %w", err)
	}
	defer elfFile.Close()

	if err := validateELF(elfFile.FileHeader, a.Arch); err != nil {
		return fmt.Errorf("check main binary: %w", err)
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
