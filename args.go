// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package main

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

// Set on build.
var (
	version = "dev"
	commit  = "none"    //nolint:gochecknoglobals
	date    = "unknown" //nolint:gochecknoglobals
)

type limitedUintFlag struct {
	value    uint
	min, max uint64
	unit     string
}

func (u limitedUintFlag) MarshalText() ([]byte, error) {
	return []byte(strconv.Itoa(int(u.value)) + u.unit), nil
}

var errValueOutsideRange = errors.New("value is outside of range")

func (u *limitedUintFlag) UnmarshalText(text []byte) error {
	value, err := strconv.ParseUint(string(text), 10, 0)
	if err != nil {
		return err
	}

	if u.min > 0 && value < u.min {
		return fmt.Errorf("%d < %d: %v", value, u.min, errValueOutsideRange)
	}

	if u.max > 0 && value > u.max {
		return fmt.Errorf("%d > %d: %v", value, u.max, errValueOutsideRange)
	}

	u.value = uint(value)

	return nil
}

type transport struct {
	qemu.TransportType
}

var errInvalidTransportType = errors.New("unknown transport type")

// MarshalText implements [encoding.TextMarshaler].
func (t transport) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnarshalText implements [encoding.TextUnmarshaler].
func (t *transport) UnmarshalText(text []byte) error {
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

type qemuArgs struct {
	qemu                string
	kernel              filePath
	machine             string
	cpu                 string
	smp                 limitedUintFlag
	memory              limitedUintFlag
	transport           transport
	initArgs            []string
	noKVM               bool
	verbose             bool
	noGoTestFlagRewrite bool
}

type initramfsArgs struct {
	arch          string
	binary        filePath
	files         filePathList
	modules       filePathList
	standalone    bool
	keepInitramfs bool
}

type args struct {
	qemuArgs
	initramfsArgs
	version bool
	debug   bool
}

func newArgs(arch string) (args, error) {
	cmd, err := qemu.NewCommand(arch)
	if err != nil {
		return args{}, err
	}

	args := args{
		qemuArgs: qemuArgs{
			qemu:    cmd.Executable,
			machine: cmd.Machine,
			cpu:     cmd.CPU,
			memory: limitedUintFlag{
				cmd.Memory,
				memMin,
				memMax,
				"MB",
			},
			smp: limitedUintFlag{
				cmd.SMP,
				smpMin,
				smpMax,
				"",
			},
			transport: transport{cmd.TransportType},
			noKVM:     cmd.NoKVM,
		},
		initramfsArgs: initramfsArgs{
			arch: arch,
		},
	}

	return args, nil
}

func (a *args) newFlagset(self string) *flag.FlagSet {
	fsName := self + " [flags...] binary [initargs...]"
	fs := flag.NewFlagSet(fsName, flag.ContinueOnError)

	fs.StringVar(
		&a.qemu,
		"qemu-bin",
		a.qemu,
		"QEMU binary to use",
	)

	fs.TextVar(
		&a.kernel,
		"kernel",
		a.kernel,
		"path to kernel to use",
	)

	fs.StringVar(
		&a.machine,
		"machine",
		a.machine,
		"QEMU machine type to use",
	)

	fs.StringVar(
		&a.cpu,
		"cpu",
		a.cpu,
		"QEMU CPU type to use",
	)

	fs.BoolVar(
		&a.noKVM,
		"nokvm",
		a.noKVM,
		"disable hardware support",
	)

	fs.TextVar(
		&a.transport,
		"transport",
		a.transport,
		"io transport type: isa/0, pci/1, mmio/2",
	)

	fs.BoolVar(
		&a.verbose,
		"verbose",
		a.verbose,
		"enable verbose guest system output",
	)

	fs.TextVar(
		&a.memory,
		"memory",
		a.memory,
		"memory (in MB) for the QEMU VM",
	)

	fs.TextVar(
		&a.smp,
		"smp",
		a.smp,
		"number of CPUs for the QEMU VM",
	)

	fs.BoolVar(
		&a.standalone,
		"standalone",
		a.standalone,
		"run first given file as init itself. Use this if it has virtrun support built in.",
	)

	fs.BoolVar(
		&a.noGoTestFlagRewrite,
		"noGoTestFlagRewrite",
		a.noGoTestFlagRewrite,
		"disable automatic go test flag rewrite for file based output.",
	)

	fs.BoolVar(
		&a.keepInitramfs,
		"keepInitramfs",
		a.keepInitramfs,
		"do not delete initramfs once qemu is done. Intended for debugging. "+
			"The path to the file is printed on stderr",
	)

	fs.Var(
		&a.files,
		"addFile",
		"file to add to guest's /data dir. Flag may be used more than once.",
	)

	fs.Var(
		&a.modules,
		"addModule",
		"kernel module to add to guest. Flag may be used more than once.",
	)

	fs.BoolVar(
		&a.version,
		"version",
		a.version,
		"show version and exit",
	)

	fs.BoolVar(
		&a.debug,
		"debug",
		a.debug,
		"enable debug output",
	)

	return fs
}

func (a *args) parseArgs(name string, args []string, output io.Writer) error {
	fs := a.newFlagset(name)
	fs.SetOutput(output)

	// Parses arguments up to the first one that is not prefixed with a "-" or
	// is "--".
	if err := fs.Parse(args); err != nil {
		return err
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

		return errors.New(msg)
	}

	// With version flag, just print the version and exit. Using [flag.ErrHelp]
	// the main binary is supposed to return with a non error exit code.
	if a.version {
		msgFmt := "virtrun %s\n  commit %s\n  built at %s"
		printf(msgFmt, version, commit, date)

		return flag.ErrHelp
	}

	if a.kernel == "" {
		return failf("no kernel given (use -kernel)")
	}

	// First positional argument is supposed to be a binary file.
	if len(fs.Args()) < 1 {
		return failf("no binary given")
	}

	binary, err := absoluteFilePath(fs.Args()[0])
	if err != nil {
		return failf("binary path: %v", err)
	}

	a.binary = binary

	// All further positional arguments after the binary file will be passed to
	// the guest system's init program.
	a.initArgs = fs.Args()[1:]

	return nil
}

func (a *args) validate() error {
	// Check files are actually present.
	if _, err := exec.LookPath(a.qemu); err != nil {
		return fmt.Errorf("check qemu binary: %v", err)
	}

	if err := a.kernel.check(); err != nil {
		return fmt.Errorf("check kernel file: %v", err)
	}

	for _, file := range a.files {
		if err := filePath(file).check(); err != nil {
			return fmt.Errorf("check file: %v", err)
		}
	}

	for _, file := range a.modules {
		if err := filePath(file).check(); err != nil {
			return fmt.Errorf("check module: %v", err)
		}
	}

	// Do some deeper validation for the main binary.
	elfFile, err := elf.Open(string(a.binary))
	if err != nil {
		return fmt.Errorf("check main binary: %v", err)
	}
	defer elfFile.Close()

	if err := validateELF(elfFile.FileHeader, a.arch); err != nil {
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
