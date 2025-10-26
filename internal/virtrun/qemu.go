// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package virtrun

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/aibor/virtrun/internal/exitcode"
	"github.com/aibor/virtrun/internal/qemu"
	"github.com/aibor/virtrun/internal/sys"
)

// Qemu specifies the input for creating a new [qemu.Command].
type Qemu struct {
	Executable          string
	Kernel              string
	Machine             string
	CPU                 string
	SMP                 uint64
	Memory              uint64
	TransportType       qemu.TransportType
	InitArgs            []string
	ExtraArgs           []qemu.Argument
	NoKVM               bool
	Verbose             bool
	NoGoTestFlagRewrite bool
}

func (s *Qemu) addDefaultsFor(arch sys.Arch) error {
	var (
		executable    string
		machine       string
		transportType qemu.TransportType
	)

	switch arch {
	case sys.AMD64:
		executable = "qemu-system-x86_64"
		machine = "q35"
		transportType = qemu.TransportTypePCI
	case sys.ARM64:
		executable = "qemu-system-aarch64"
		machine = "virt"
		transportType = qemu.TransportTypeMMIO
	case sys.RISCV64:
		executable = "qemu-system-riscv64"
		machine = "virt"
		transportType = qemu.TransportTypeMMIO
	default:
		return sys.ErrArchNotSupported
	}

	if s.Executable == "" {
		s.Executable = executable
	}

	if s.Machine == "" {
		s.Machine = machine
	}

	if s.TransportType == "" {
		s.TransportType = transportType
	}

	if !s.NoKVM {
		s.NoKVM = !arch.KVMAvailable()
	}

	return nil
}

// NewQemuCommand creates a new [qemu.Command] with the given spec and
// initramfs.
func NewQemuCommand(cfg Qemu, initramfsPath string) (*qemu.Command, error) {
	cmdSpec := qemu.CommandSpec{
		Executable:     cfg.Executable,
		Kernel:         cfg.Kernel,
		Initramfs:      initramfsPath,
		Machine:        cfg.Machine,
		CPU:            cfg.CPU,
		Memory:         cfg.Memory,
		SMP:            cfg.SMP,
		TransportType:  cfg.TransportType,
		InitArgs:       cfg.InitArgs,
		ExtraArgs:      cfg.ExtraArgs,
		NoKVM:          cfg.NoKVM,
		Verbose:        cfg.Verbose,
		ExitCodeParser: exitcode.Parse,
	}

	// In order to be useful with "go test -exec", rewrite the file based flags
	// so the output can be passed from guest to kernel via consoles.
	if !cfg.NoGoTestFlagRewrite {
		rewriteGoTestFlagsPath(&cmdSpec)
	}

	cmd, err := qemu.NewCommand(cmdSpec)
	if err != nil {
		return nil, fmt.Errorf("build command: %w", err)
	}

	slog.Debug("QEMU command", slog.String("command", cmd.String()))

	return cmd, nil
}

// rewriteGoTestFlagsPath processes file related go test flags in
// [qemu.CommandSpec.InitArgs] and changes them, so the guest system's writes
// end up in the host systems file paths.
//
// It scans [qemu.CommandSpec.InitArgs] for coverage and profile related paths
// and replaces them with console path. The original paths are added as
// additional file descriptors to the [qemu.CommandSpec].
//
// It is required that the flags are prefixed with "test" and value is
// separated form the flag by "=". This is the format the "go test" tool
// invokes the test binary with.
func rewriteGoTestFlagsPath(spec *qemu.CommandSpec) {
	const splitNum = 2

	outputDir := ""

	for idx, posArg := range spec.InitArgs {
		splits := strings.SplitN(posArg, "=", splitNum)
		switch splits[0] {
		case "-test.outputdir":
			outputDir = splits[1]
			fallthrough
		case "-test.gocoverdir":
			splits[1] = "/tmp"
		}

		spec.InitArgs[idx] = strings.Join(splits, "=")
	}

	// Only coverprofile has a relative path to the test pwd and can be
	// replaced immediately. All other profile files are relative to the actual
	// test running and need to be prefixed with -test.outputdir. So, collect
	// them and process them afterwards when "outputdir" is found.
	for idx, posArg := range spec.InitArgs {
		splits := strings.SplitN(posArg, "=", splitNum)
		switch splits[0] {
		case "-test.blockprofile",
			"-test.cpuprofile",
			"-test.memprofile",
			"-test.mutexprofile",
			"-test.trace":
			if !filepath.IsAbs(splits[1]) {
				splits[1] = filepath.Join(outputDir, splits[1])
			}

			fallthrough
		case "-test.coverprofile":
			splits[1] = spec.AddConsole(splits[1])
		}

		spec.InitArgs[idx] = strings.Join(splits, "=")
	}
}
