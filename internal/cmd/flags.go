// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"flag"
	"fmt"
	"io"
	"runtime/debug"

	"github.com/aibor/virtrun/internal/virtrun"
)

// Set on build.
var version = "dev"

type Flags struct {
	name string
	spec *virtrun.Spec

	versionFlag bool
	debugFlag   bool
	flagSet     *flag.FlagSet
}

func NewFlags(name string, spec *virtrun.Spec, output io.Writer) *Flags {
	flags := &Flags{
		name: name,
		spec: spec,
	}

	flags.initFlagset(output)

	return flags
}

func (f *Flags) initFlagset(output io.Writer) {
	fsName := f.name + " [flags...] binary [initargs...]"
	fs := flag.NewFlagSet(fsName, flag.ContinueOnError)
	fs.SetOutput(output)

	fs.StringVar(
		&f.spec.Qemu.Executable,
		"qemu-bin",
		f.spec.Qemu.Executable,
		"QEMU binary to use",
	)

	fs.TextVar(
		&f.spec.Qemu.Kernel,
		"kernel",
		&f.spec.Qemu.Kernel,
		"path to kernel to use",
	)

	fs.StringVar(
		&f.spec.Qemu.Machine,
		"machine",
		f.spec.Qemu.Machine,
		"QEMU machine type to use",
	)

	fs.StringVar(
		&f.spec.Qemu.CPU,
		"cpu",
		f.spec.Qemu.CPU,
		"QEMU CPU type to use",
	)

	fs.BoolVar(
		&f.spec.Qemu.NoKVM,
		"nokvm",
		f.spec.Qemu.NoKVM,
		"disable hardware support",
	)

	fs.TextVar(
		&f.spec.Qemu.TransportType,
		"transport",
		&f.spec.Qemu.TransportType,
		"io transport type: isa, pci, mmio",
	)

	fs.BoolVar(
		&f.spec.Qemu.Verbose,
		"verbose",
		f.spec.Qemu.Verbose,
		"enable verbose guest system output",
	)

	fs.TextVar(
		&f.spec.Qemu.Memory,
		"memory",
		&f.spec.Qemu.Memory,
		"memory (in MB) for the QEMU VM",
	)

	fs.TextVar(
		&f.spec.Qemu.SMP,
		"smp",
		&f.spec.Qemu.SMP,
		"number of CPUs for the QEMU VM",
	)

	fs.BoolVar(
		&f.spec.Initramfs.StandaloneInit,
		"standalone",
		f.spec.Initramfs.StandaloneInit,
		"run first given file as init itself. Use this if it has virtrun"+
			" support built in.",
	)

	fs.BoolVar(
		&f.spec.Qemu.NoGoTestFlagRewrite,
		"noGoTestFlagRewrite",
		f.spec.Qemu.NoGoTestFlagRewrite,
		"disable automatic go test flag rewrite for file based output.",
	)

	fs.BoolVar(
		&f.spec.Initramfs.Keep,
		"keepInitramfs",
		f.spec.Initramfs.Keep,
		"do not delete initramfs once qemu is done. Intended for debugging. "+
			"The path to the file is printed on stderr",
	)

	fs.Var(
		&f.spec.Initramfs.Files,
		"addFile",
		"file to add to guest's /data dir. Flag may be used more than once.",
	)

	fs.Var(
		&f.spec.Initramfs.Modules,
		"addModule",
		"kernel module to add to guest. Flag may be used more than once.",
	)

	fs.BoolVar(
		&f.debugFlag,
		"debug",
		f.debugFlag,
		"enable debug output",
	)

	fs.BoolVar(
		&f.versionFlag,
		"version",
		f.versionFlag,
		"show version and exit",
	)

	f.flagSet = fs
}

// Fail fails like flag does. It prints the error first and then usage.
func (f *Flags) Fail(msg string, err error) error {
	err = &ParseArgsError{msg: msg, err: err}
	fmt.Fprintln(f.flagSet.Output(), err.Error())

	f.flagSet.Usage()

	return err
}

func (f *Flags) Debug() bool {
	return f.debugFlag
}

func (f *Flags) printVersionInformation() {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	fmt.Fprintf(f.flagSet.Output(), "%s: %s\n\n", f.name, version)
	fmt.Fprintln(f.flagSet.Output(), buildInfo.String())
}

func (f *Flags) ParseArgs(args []string) error {
	// Parses arguments up to the first one that is not prefixed with a "-" or
	// is "--".
	if err := f.flagSet.Parse(args); err != nil {
		return &ParseArgsError{msg: "flag parse: %w", err: err}
	}

	// With version flag, just print the version and exit. Using [flag.ErrHelp]
	// the main binary is supposed to return with a non error exit code.
	if f.versionFlag {
		f.printVersionInformation()
		return &ParseArgsError{msg: "version requested", err: flag.ErrHelp}
	}

	if f.spec.Qemu.Kernel == "" {
		return f.Fail("no kernel given (use -kernel)", nil)
	}

	positionalArgs := f.flagSet.Args()

	// First positional argument is supposed to be a binary file.
	if len(positionalArgs) < 1 {
		return f.Fail("no binary given", nil)
	}

	binary, err := virtrun.AbsoluteFilePath(positionalArgs[0])
	if err != nil {
		return f.Fail("binary path", err)
	}

	f.spec.Initramfs.Binary = binary

	// All further positional arguments after the binary file will be passed to
	// the guest system's init program.
	f.spec.Qemu.InitArgs = positionalArgs[1:]

	return nil
}
