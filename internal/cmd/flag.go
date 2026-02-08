// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"flag"
	"fmt"
	"io"
	"log/slog"

	"github.com/aibor/virtrun/internal/qemu"
	"github.com/aibor/virtrun/internal/sys"
)

const (
	name = "virtrun"

	cpuDefault = "max"

	memDefault = 256
	memMin     = 128
	memMax     = 16384

	smpDefault = 1
	smpMin     = 1
	smpMax     = 16

	usageMessage = `Usage of 'virtrun':
    virtrun [flags...] binary [initargs...]

Using it directly:
	virtrun -kernel=/path/to/kernel ./my_binary -flagForBinary=3

Using it with go test:
	go test -exec 'virtrun -kernel=/path/to/kernel' ./...

All virtrun flags can also be provided via environment variable VIRTRUN_ARGS:
	VIRTRUN_ARGS="-kernel=/path/to/kernel -debug" go test -exec virtrun ./...

All virtrun flags can also be provided via file ./virtrun-args, with one
argument per line.
`
)

type flagSet struct {
	*flag.FlagSet
}

func newFlagSet(name string, flags *flags) *flagSet {
	flagSet := flagSet{flag.NewFlagSet(name, flag.ContinueOnError)}

	flagSet.StringVar(&flags.QemuBin, "qemuBin", flags.QemuBin,
		"QEMU binary to use (default depends on binary arch: qemu-system-*)")

	flagSet.FilePath(&flags.KernelPath, "kernel",
		"path to kernel to use")

	flagSet.StringVar(&flags.Machine, "machine", flags.Machine,
		"QEMU machine type to use (default depends on binary arch)")

	flagSet.StringVar(&flags.CPUType, "cpu", flags.CPUType,
		"QEMU CPU type to use")

	flagSet.BoolVar(&flags.NoKVM, "nokvm", flags.NoKVM,
		"disable hardware support (default is enabled if present and binary "+
			"matches the host arch)")

	flagSet.Var(&flags.TransportType, "transport",
		"io transport type: isa, pci, mmio (default depends on binary arch)")

	flagSet.BoolVar(&flags.GuestVerbose, "verbose", flags.GuestVerbose,
		"enable verbose guest system output")

	flagSet.LimitedUintVar(&flags.Memory, memMin, memMax, "memory",
		"memory (in MB) for the QEMU VM")

	flagSet.LimitedUintVar(&flags.NumCPU, smpMin, smpMax, "smp",
		"number of CPUs for the QEMU VM")

	flagSet.BoolVar(&flags.Standalone, "standalone", flags.Standalone,
		"run binary as init itself (must have virtrun supprot built in)")

	flagSet.BoolVar(&flags.NoGoTestFlags, "noGoTestFlagRewrite",
		flags.NoGoTestFlags,
		"disable automatic go test flag rewrite for file based output.")

	flagSet.BoolVar(&flags.KeepInitramfs, "keepInitramfs", flags.KeepInitramfs,
		"do not delete initramfs on exit. Intended for debugging. "+
			"The path to the file is printed on stderr")

	flagSet.FilePathList(&flags.DataFilePaths, "addFile",
		"file to add to guest's /data dir. Flag may be used more than once. "+
			"Empty value clears the list.")

	flagSet.FilePathList(&flags.ModulePaths, "addModule",
		"kernel module to add to guest. Flag may be used more than once. "+
			"Empty value clears the list.")

	flagSet.BoolVar(&flags.Debug, "debug", flags.Debug,
		"enable debug output")

	flagSet.BoolVar(&flags.Version, "version", flags.Version,
		"show version and exit")

	return &flagSet
}

func (f *flagSet) LimitedUintVar(
	value *uint64,
	lower, upper uint64,
	name string,
	usage string,
) {
	flagValue := LimitedUintValue{
		Value: value,
		Lower: lower,
		Upper: upper,
	}
	f.Var(&flagValue, name, usage)
}

func (f *flagSet) FilePath(value *string, name string, usage string) {
	f.Var((*FilePath)(value), name, usage)
}

func (f *flagSet) FilePathList(value *[]string, name string, usage string) {
	f.Var((*FilePathList)(value), name, usage)
}

// fail fails like flag does. It prints the error first and then usage.
func (f *flagSet) fail(msg string, err error) error {
	err = &ParseArgsError{msg: msg, err: err}
	fmt.Fprintln(f.Output(), err.Error())

	f.Usage()

	return err
}

// flags is a collection of all defined command flags.
type flags struct {
	QemuBin        string
	CPUType        string
	Machine        string
	Memory         uint64
	NumCPU         uint64
	TransportType  qemu.TransportType
	KernelPath     string
	ExecutablePath string
	DataFilePaths  []string
	ModulePaths    []string
	InitArgs       []string
	Standalone     bool
	KeepInitramfs  bool
	NoKVM          bool
	GuestVerbose   bool
	NoGoTestFlags  bool
	Version        bool
	Debug          bool
}

// parseArgs parses the given argument list into [flags]. Additional error
// output is written to the given writer.
func parseArgs(args []string, output io.Writer) (*flags, error) {
	flags := &flags{
		CPUType: cpuDefault,
		Memory:  memDefault,
		NumCPU:  smpDefault,
	}

	flagSet := newFlagSet(name, flags)
	flagSet.SetOutput(output)
	flagSet.Usage = func() {
		fmt.Fprint(output, usageMessage)
		fmt.Fprintln(output, "\nFlags:")
		flagSet.PrintDefaults()
	}

	// Parses arguments up to the first one that is not prefixed with a "-" or
	// is "--".
	err := flagSet.Parse(args)
	if err != nil {
		return nil, &ParseArgsError{msg: "flag parse: %w", err: err}
	}

	// With version flag, just print the version and exit.
	if flags.Version {
		return flags, nil
	}

	if flags.KernelPath == "" {
		return nil, flagSet.fail("no kernel given (use -kernel)", nil)
	}

	positionalArgs := flagSet.Args()

	// First positional argument is supposed to be a binary file.
	if len(positionalArgs) < 1 {
		return nil, flagSet.fail("no binary given", nil)
	}

	flags.ExecutablePath, err = sys.AbsolutePath(positionalArgs[0])
	if err != nil {
		return nil, flagSet.fail("binary path", err)
	}

	// All further positional arguments after the binary file will be passed to
	// the guest system's init program.
	flags.InitArgs = positionalArgs[1:]

	return flags, nil
}

func (f *flags) logLevel() slog.Level {
	if f.Debug {
		return slog.LevelDebug
	}

	return slog.LevelWarn
}

func (f *flags) validateFilePaths() error {
	err := ValidateFilePath(f.KernelPath)
	if err != nil {
		return fmt.Errorf("kernel file: %w", err)
	}

	for _, file := range f.DataFilePaths {
		err := ValidateFilePath(file)
		if err != nil {
			return fmt.Errorf("additional file: %w", err)
		}
	}

	for _, file := range f.ModulePaths {
		err := ValidateFilePath(file)
		if err != nil {
			return fmt.Errorf("module: %w", err)
		}
	}

	err = ValidateFilePath(f.ExecutablePath)
	if err != nil {
		return fmt.Errorf("main binary: %w", err)
	}

	return nil
}
