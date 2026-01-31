// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"flag"
	"fmt"
	"io"
	"runtime/debug"

	"github.com/aibor/virtrun/internal/sys"
	"github.com/aibor/virtrun/internal/virtrun"
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

type flags struct {
	spec    virtrun.Spec
	flagSet *flag.FlagSet

	version             bool
	debug               bool
	noGoTestFlagRewrite bool
}

func newFlags(output io.Writer) *flags {
	flags := &flags{
		spec: virtrun.Spec{
			Qemu: virtrun.Qemu{
				CPU:    cpuDefault,
				Memory: memDefault,
				SMP:    smpDefault,
			},
		},
	}

	flags.initFlagset(output)

	return flags
}

func (f *flags) ParseArgs(args []string) error {
	// Parses arguments up to the first one that is not prefixed with a "-" or
	// is "--".
	err := f.flagSet.Parse(args)
	if err != nil {
		return &ParseArgsError{msg: "flag parse: %w", err: err}
	}

	// With version flag, just print the version and exit. Using [ErrHelp]
	// the main binary is supposed to return with a non error exit code.
	if f.version {
		err := f.printVersionInformation()
		return &ParseArgsError{msg: "version requested", err: err}
	}

	if f.spec.Qemu.Kernel == "" {
		return f.fail("no kernel given (use -kernel)", nil)
	}

	positionalArgs := f.flagSet.Args()

	// First positional argument is supposed to be a binary file.
	if len(positionalArgs) < 1 {
		return f.fail("no binary given", nil)
	}

	binary, err := sys.AbsolutePath(positionalArgs[0])
	if err != nil {
		return f.fail("binary path", err)
	}

	f.spec.Initramfs.Binary = binary

	// All further positional arguments after the binary file will be passed to
	// the guest system's init program.
	f.spec.Qemu.InitArgs = positionalArgs[1:]

	// In order to be useful with "go test -exec", rewrite the file based flags
	// so the output can be passed from guest to kernel via consoles.
	if !f.noGoTestFlagRewrite {
		initArgs, files := virtrun.RewriteGoTestFlagsPath(positionalArgs[1:])
		f.spec.Qemu.InitArgs = initArgs
		f.spec.Qemu.AdditionalOutputFiles = files
	}

	return nil
}

func (f *flags) initFlagset(output io.Writer) {
	flagSet := flag.NewFlagSet(name, flag.ContinueOnError)
	flagSet.SetOutput(output)
	flagSet.Usage = f.usage

	flagSet.StringVar(
		&f.spec.Qemu.Executable,
		"qemuBin",
		f.spec.Qemu.Executable,
		"QEMU binary to use (default depends on binary arch: qemu-system-*)",
	)

	flagSet.Var(
		(*FilePath)(&f.spec.Qemu.Kernel),
		"kernel",
		"path to kernel to use",
	)

	flagSet.StringVar(
		&f.spec.Qemu.Machine,
		"machine",
		f.spec.Qemu.Machine,
		"QEMU machine type to use (default depends on binary arch)",
	)

	flagSet.StringVar(
		&f.spec.Qemu.CPU,
		"cpu",
		f.spec.Qemu.CPU,
		"QEMU CPU type to use",
	)

	flagSet.BoolVar(
		&f.spec.Qemu.NoKVM,
		"nokvm",
		f.spec.Qemu.NoKVM,
		"disable hardware support (default is enabled if present and binary "+
			"matches the host arch)",
	)

	flagSet.Var(
		&f.spec.Qemu.TransportType,
		"transport",
		"io transport type: isa, pci, mmio (default depends on binary arch)",
	)

	flagSet.BoolVar(
		&f.spec.Qemu.Verbose,
		"verbose",
		f.spec.Qemu.Verbose,
		"enable verbose guest system output",
	)

	flagSet.Var(
		&limitedUintValue{
			Value: &f.spec.Qemu.Memory,
			min:   memMin,
			max:   memMax,
		},
		"memory",
		"memory (in MB) for the QEMU VM",
	)

	flagSet.Var(
		&limitedUintValue{
			Value: &f.spec.Qemu.SMP,
			min:   smpMin,
			max:   smpMax,
		},
		"smp",
		"number of CPUs for the QEMU VM",
	)

	flagSet.BoolVar(
		&f.spec.Initramfs.StandaloneInit,
		"standalone",
		f.spec.Initramfs.StandaloneInit,
		"run binary as init itself (must have virtrun supprot built in)",
	)

	flagSet.BoolVar(
		&f.noGoTestFlagRewrite,
		"noGoTestFlagRewrite",
		f.noGoTestFlagRewrite,
		"disable automatic go test flag rewrite for file based output.",
	)

	flagSet.BoolVar(
		&f.spec.Initramfs.Keep,
		"keepInitramfs",
		f.spec.Initramfs.Keep,
		"do not delete initramfs on exit. Intended for debugging. "+
			"The path to the file is printed on stderr",
	)

	flagSet.Var(
		(*FilePathList)(&f.spec.Initramfs.Files),
		"addFile",
		"file to add to guest's /data dir. Flag may be used more than once. "+
			"Empty value clears the list.",
	)

	flagSet.Var(
		(*FilePathList)(&f.spec.Initramfs.Modules),
		"addModule",
		"kernel module to add to guest. Flag may be used more than once. "+
			"Empty value clears the list.",
	)

	flagSet.BoolVar(
		&f.debug,
		"debug",
		f.debug,
		"enable debug output",
	)

	flagSet.BoolVar(
		&f.version,
		"version",
		f.version,
		"show version and exit",
	)

	f.flagSet = flagSet
}

// fail fails like flag does. It prints the error first and then usage.
func (f *flags) fail(msg string, err error) error {
	err = &ParseArgsError{msg: msg, err: err}
	fmt.Fprintln(f.flagSet.Output(), err.Error())

	f.flagSet.Usage()

	return err
}

func (f *flags) printVersionInformation() error {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return ErrReadBuildInfo
	}

	fmt.Fprintf(f.flagSet.Output(), "Version: %s\n", buildInfo.Main.Version)

	return ErrHelp
}

func (f *flags) usage() {
	fmt.Fprint(f.flagSet.Output(), usageMessage)
	fmt.Fprintln(f.flagSet.Output(), "\nFlags:")
	f.flagSet.PrintDefaults()
}
