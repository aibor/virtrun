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
	Executable            string
	Kernel                string
	Machine               string
	CPU                   string
	SMP                   uint64
	Memory                uint64
	TransportType         qemu.TransportType
	InitArgs              []string
	ExtraArgs             []qemu.Argument
	AdditionalOutputFiles []string
	NoKVM                 bool
	Verbose               bool
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
func NewQemuCommand(spec Qemu, initramfsPath string) (*qemu.Command, error) {
	cmd, err := qemu.NewCommand(qemu.CommandSpec{
		Executable:         spec.Executable,
		Kernel:             spec.Kernel,
		Initramfs:          initramfsPath,
		Machine:            spec.Machine,
		CPU:                spec.CPU,
		Memory:             spec.Memory,
		SMP:                spec.SMP,
		TransportType:      spec.TransportType,
		InitArgs:           spec.InitArgs,
		AdditionalConsoles: spec.AdditionalOutputFiles,
		ExtraArgs:          spec.ExtraArgs,
		NoKVM:              spec.NoKVM,
		Verbose:            spec.Verbose,
	}, exitcode.Parse)
	if err != nil {
		return nil, fmt.Errorf("build command: %w", err)
	}

	slog.Debug("QEMU command", slog.String("command", cmd.String()))

	return cmd, nil
}

// RewriteGoTestFlagsPath processes file related go test flags so their file
// path are correct for use in the guest system.
//
// It is required that the flags are prefixed with "test" and value is
// separated form the flag by "=". This is the format the "go test" tool
// invokes the test binary with.
//
// Each file path is replaced with a path to a serial console. The modified args
// are returned along with a list of the host file paths.
//
//revive:disable:confusing-results
func RewriteGoTestFlagsPath(args []string) ([]string, []string) {
	const splitNum = 2

	outputDir := ""
	outputArgs := make([]string, len(args))

	for idx, posArg := range args {
		splits := strings.SplitN(posArg, "=", splitNum)
		switch splits[0] {
		case "-test.outputdir":
			outputDir = splits[1]
			fallthrough
		case "-test.gocoverdir":
			splits[1] = "/tmp"
		}

		outputArgs[idx] = strings.Join(splits, "=")
	}

	var files []string

	// Only coverprofile has a relative path to the test pwd and can be
	// replaced immediately. All other profile files are relative to the actual
	// test running and need to be prefixed with -test.outputdir. So, collect
	// them and process them afterwards when "outputdir" is found.
	for idx, posArg := range outputArgs {
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
			files = append(files, splits[1])
			splits[1] = qemu.AdditionalConsolePath(len(files) - 1)
		}

		outputArgs[idx] = strings.Join(splits, "=")
	}

	return outputArgs, files
}

//revive:enable:confusing-results
