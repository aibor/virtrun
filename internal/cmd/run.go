// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"runtime/debug"

	"github.com/aibor/virtrun/internal/exitcode"
	"github.com/aibor/virtrun/internal/initramfs"
	"github.com/aibor/virtrun/internal/qemu"
	"github.com/aibor/virtrun/internal/sys"
)

const localConfigFile = ".virtrun-args"

const archivePrefix = "virtrun-initramfs"

// IO provides input and output details for the command.
type IO struct {
	Stdin  *os.File
	Stdout *os.File
	Stderr *os.File
}

func newFlags(args []string, cfg IO) (*flags, error) {
	args, err := MergedArgs(args, os.DirFS("."), localConfigFile)
	if err != nil {
		return nil, err
	}

	flags, err := parseArgs(args, cfg.Stderr)
	if err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}

	return flags, nil
}

func newInitramfs(
	ctx context.Context,
	flags *flags,
	arch sys.Arch,
) (*initramfs.FS, error) {
	var initProg fs.File

	// In standalone mode, the main file is supposed to work as a complete
	// init matching our requirements.
	if !flags.Standalone {
		var err error

		initProg, err = initramfs.InitProgFor(arch)
		if err != nil {
			return nil, fmt.Errorf("get init program: %w", err)
		}
	}

	initramfsSpec := initramfs.Spec{
		Executable: flags.ExecutablePath,
		Files:      flags.DataFilePaths,
		Modules:    flags.ModulePaths,
		Fsys:       os.DirFS("/"),
		Init:       initProg,
	}

	initramFS, err := initramfs.New(ctx, initramfsSpec)
	if err != nil {
		return nil, fmt.Errorf("build initramfs: %w", err)
	}

	return initramFS, nil
}

func newQemuCommand(
	flags *flags,
	arch sys.Arch,
	initramfsPath string,
) (*qemu.Command, error) {
	qemuSpec := qemu.CommandSpec{
		Executable:    flags.QemuBin,
		Kernel:        flags.KernelPath,
		Initramfs:     initramfsPath,
		Machine:       flags.Machine,
		CPU:           flags.CPUType,
		SMP:           flags.NumCPU,
		Memory:        flags.Memory,
		TransportType: flags.TransportType,
		InitArgs:      flags.InitArgs,
		NoKVM:         flags.NoKVM,
		Verbose:       flags.GuestVerbose,
	}

	err := qemuSpec.AddDefaultsFor(arch)
	if err != nil {
		return nil, fmt.Errorf("qemu defaults: %w", err)
	}

	// In order to be useful with "go test -exec", rewrite the file based flags
	// so the output can be passed from guest to kernel via consoles.
	if !flags.NoGoTestFlags {
		qemuSpec.RewriteGoTestFlagsPath()
	}

	cmd, err := qemu.NewCommand(qemuSpec, exitcode.Parse)
	if err != nil {
		return nil, fmt.Errorf("new qemu command: %w", err)
	}

	return cmd, nil
}

func run(ctx context.Context, flags *flags, cfg IO) error {
	err := flags.validateFilePaths()
	if err != nil {
		return fmt.Errorf("validate: %w", err)
	}

	arch, err := sys.ReadELFArch(flags.ExecutablePath)
	if err != nil {
		return fmt.Errorf("read main executable arch: %w", err)
	}

	initFS, err := newInitramfs(ctx, flags, arch)
	if err != nil {
		return err
	}

	initFSPath, err := initramfs.WriteToTempFile(initFS, "", archivePrefix)
	if err != nil {
		return fmt.Errorf("write initramfs: %w", err)
	}

	slog.Debug("Created initramfs archive",
		slog.String("path", initFSPath))

	cmd, err := newQemuCommand(flags, arch, initFSPath)
	if err != nil {
		_ = os.Remove(initFSPath)
		return err
	}

	slog.Debug("QEMU command",
		slog.String("command", cmd.String()))

	if flags.KeepInitramfs {
		defer slog.Info("Preserving initramfs archive",
			slog.String("path", initFSPath))
	} else {
		defer removeInitramfs(initFSPath)
	}

	err = cmd.Run(ctx, cfg.Stdin, cfg.Stdout, cfg.Stderr)
	if err != nil {
		return fmt.Errorf("qemu: %w", err)
	}

	return nil
}

func removeInitramfs(path string) {
	slog.Debug("Removing initramfs archive", slog.String("path", path))

	err := os.Remove(path)
	if err != nil {
		slog.Error(
			"Failed to remove initramfs archive",
			slog.String("path", path),
			slog.Any("error", err),
		)
	}
}

func handleParseArgsError(err error) int {
	// [ErrHelp] is returned when help is requested. So exit without error
	// in this case.
	if errors.Is(err, ErrHelp) {
		return 0
	}

	// ParseArgs already prints errors, so we just exit without an error.
	if !errors.Is(err, &ParseArgsError{}) {
		slog.Error(err.Error())
	}

	return -1
}

func handleRunError(err error) int {
	exitCode := -1

	var qemuErr *qemu.CommandError
	if errors.As(err, &qemuErr) {
		if qemuErr.ExitCode != 0 {
			exitCode = qemuErr.ExitCode
		}
	}

	// Do not print the error in case the guest process ran successfully and
	// the guest properly communicated a non-zero exit code.
	if !errors.Is(err, qemu.ErrGuestNonZeroExitCode) {
		slog.Error(err.Error())
	}

	return exitCode
}

// Run is the main entry point for the CLI command.
func Run(ctx context.Context, args []string, cfg IO) int {
	log.SetOutput(cfg.Stderr)
	log.SetFlags(log.Lmicroseconds)
	log.SetPrefix("VIRTRUN: ")

	flags, err := newFlags(args, cfg)
	if err != nil {
		return handleParseArgsError(err)
	}

	slog.SetLogLoggerLevel(flags.logLevel())

	if flags.Version {
		buildInfo, err := getBuildInfo()
		if err != nil {
			slog.Error(err.Error())
			return -1
		}

		fmt.Fprintf(cfg.Stdout, "Version: %s\n", buildInfo.Main.Version)

		return 0
	}

	err = run(ctx, flags, cfg)
	if err != nil {
		return handleRunError(err)
	}

	return 0
}

func getBuildInfo() (*debug.BuildInfo, error) {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return nil, ErrReadBuildInfo
	}

	return buildInfo, nil
}
