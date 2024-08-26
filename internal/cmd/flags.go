// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"flag"
	"fmt"
	"io"

	"github.com/aibor/virtrun/internal/sys"
	"github.com/aibor/virtrun/internal/virtrun"
)

// Set on build.
var (
	version = "dev"
	commit  = "none"    //nolint:gochecknoglobals
	date    = "unknown" //nolint:gochecknoglobals
)

func newFlagset(cfg *virtrun.Virtrun, self string) *flag.FlagSet {
	fsName := self + " [flags...] binary [initargs...]"
	fs := flag.NewFlagSet(fsName, flag.ContinueOnError)

	fs.StringVar(
		&cfg.Qemu.Executable,
		"qemu-bin",
		cfg.Qemu.Executable,
		"QEMU binary to use",
	)

	fs.TextVar(
		&cfg.Qemu.Kernel,
		"kernel",
		cfg.Qemu.Kernel,
		"path to kernel to use",
	)

	fs.StringVar(
		&cfg.Qemu.Machine,
		"machine",
		cfg.Qemu.Machine,
		"QEMU machine type to use",
	)

	fs.StringVar(
		&cfg.Qemu.CPU,
		"cpu",
		cfg.Qemu.CPU,
		"QEMU CPU type to use",
	)

	fs.BoolVar(
		&cfg.Qemu.NoKVM,
		"nokvm",
		cfg.Qemu.NoKVM,
		"disable hardware support",
	)

	fs.TextVar(
		&cfg.Qemu.TransportType,
		"transport",
		cfg.Qemu.TransportType,
		"io transport type: isa, pci, mmio",
	)

	fs.BoolVar(
		&cfg.Qemu.Verbose,
		"verbose",
		cfg.Qemu.Verbose,
		"enable verbose guest system output",
	)

	fs.TextVar(
		&cfg.Qemu.Memory,
		"memory",
		cfg.Qemu.Memory,
		"memory (in MB) for the QEMU VM",
	)

	fs.TextVar(
		&cfg.Qemu.SMP,
		"smp",
		cfg.Qemu.SMP,
		"number of CPUs for the QEMU VM",
	)

	fs.BoolVar(
		&cfg.Initramfs.StandaloneInit,
		"standalone",
		cfg.Initramfs.StandaloneInit,
		"run first given file as init itself. Use this if it has virtrun"+
			" support built in.",
	)

	fs.BoolVar(
		&cfg.Qemu.NoGoTestFlagRewrite,
		"noGoTestFlagRewrite",
		cfg.Qemu.NoGoTestFlagRewrite,
		"disable automatic go test flag rewrite for file based output.",
	)

	fs.BoolVar(
		&cfg.Initramfs.Keep,
		"keepInitramfs",
		cfg.Initramfs.Keep,
		"do not delete initramfs once qemu is done. Intended for debugging. "+
			"The path to the file is printed on stderr",
	)

	fs.Var(
		&cfg.Initramfs.Files,
		"addFile",
		"file to add to guest's /data dir. Flag may be used more than once.",
	)

	fs.Var(
		&cfg.Initramfs.Modules,
		"addModule",
		"kernel module to add to guest. Flag may be used more than once.",
	)

	fs.BoolVar(
		&cfg.Version,
		"version",
		cfg.Version,
		"show version and exit",
	)

	fs.BoolVar(
		&cfg.Debug,
		"debug",
		cfg.Debug,
		"enable debug output",
	)

	return fs
}

func ParseArgs(
	cfg *virtrun.Virtrun,
	name string,
	args []string,
	output io.Writer,
) error {
	fs := newFlagset(cfg, name)
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
	if cfg.Version {
		msgFmt := "virtrun %s\n  commit %s\n  built at %s"
		printf(msgFmt, version, commit, date)

		return &ParseArgsError{err: flag.ErrHelp}
	}

	if cfg.Qemu.Kernel == "" {
		return failf("no kernel given (use -kernel)")
	}

	// First positional argument is supposed to be a binary file.
	if len(fs.Args()) < 1 {
		return failf("no binary given")
	}

	binary, err := sys.AbsoluteFilePath(fs.Args()[0])
	if err != nil {
		return failf("binary path: %w", err)
	}

	cfg.Initramfs.Binary = binary

	// All further positional arguments after the binary file will be passed to
	// the guest system's init program.
	cfg.Qemu.InitArgs = fs.Args()[1:]

	return nil
}
