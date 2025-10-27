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
	cpuDefault = "max"

	memDefault = 256
	memMin     = 128
	memMax     = 16384

	smpDefault = 1
	smpMin     = 1
	smpMax     = 16
)

type flags struct {
	name string

	spec    virtrun.Spec
	flagSet *flag.FlagSet
	version bool
	debug   bool
}

func newFlags(name string, output io.Writer) *flags {
	flags := &flags{
		name: name,
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
	if err := f.flagSet.Parse(args); err != nil {
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

	return nil
}

func (f *flags) initFlagset(output io.Writer) {
	fsName := f.name + " [flags...] binary [initargs...]"
	flagSet := flag.NewFlagSet(fsName, flag.ContinueOnError)
	flagSet.SetOutput(output)

	flagSet.StringVar(
		&f.spec.Qemu.Executable,
		"qemuBin",
		f.spec.Qemu.Executable,
		"QEMU binary to use (default depends on binary arch)",
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
		"disable hardware support (default depends on binary arch)",
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
		"run first given file as init itself. Use this if it has virtrun"+
			" support built in.",
	)

	flagSet.BoolVar(
		&f.spec.Qemu.NoGoTestFlagRewrite,
		"noGoTestFlagRewrite",
		f.spec.Qemu.NoGoTestFlagRewrite,
		"disable automatic go test flag rewrite for file based output.",
	)

	flagSet.BoolVar(
		&f.spec.Initramfs.Keep,
		"keepInitramfs",
		f.spec.Initramfs.Keep,
		"do not delete initramfs once qemu is done. Intended for debugging. "+
			"The path to the file is printed on stderr",
	)

	flagSet.Var(
		(*FilePathList)(&f.spec.Initramfs.Files),
		"addFile",
		"file to add to guest's /data dir. Flag may be used more than once.",
	)

	flagSet.Var(
		(*FilePathList)(&f.spec.Initramfs.Modules),
		"addModule",
		"kernel module to add to guest. Flag may be used more than once.",
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
