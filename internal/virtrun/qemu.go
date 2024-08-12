// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package virtrun

import (
	"context"
	"fmt"

	"github.com/aibor/virtrun/internal/qemu"
	"github.com/aibor/virtrun/sysinit"
)

type Qemu struct {
	Executable          string
	Kernel              FilePath
	Machine             string
	CPU                 string
	SMP                 LimitedUintFlag
	Memory              LimitedUintFlag
	TransportType       qemu.TransportType
	InitArgs            []string
	ExtraArgs           []qemu.Argument
	NoKVM               bool
	Verbose             bool
	NoGoTestFlagRewrite bool
}

func NewQemuCommand(
	ctx context.Context,
	cfg Qemu,
	initramfsPath string,
) (*qemu.Command, error) {
	spec := qemu.CommandSpec{
		Executable:    cfg.Executable,
		Kernel:        string(cfg.Kernel),
		Initramfs:     initramfsPath,
		Machine:       cfg.Machine,
		CPU:           cfg.CPU,
		Memory:        cfg.Memory.Value,
		SMP:           cfg.SMP.Value,
		TransportType: cfg.TransportType,
		InitArgs:      cfg.InitArgs,
		ExtraArgs:     cfg.ExtraArgs,
		NoKVM:         cfg.NoKVM,
		Verbose:       cfg.Verbose,
		ExitCodeFmt:   sysinit.ExitCodeFmt,
	}

	// In order to be useful with "go test -exec", rewrite the file based flags
	// so the output can be passed from guest to kernel via consoles.
	if !cfg.NoGoTestFlagRewrite {
		spec.ProcessGoTestFlags()
	}

	cmd, err := qemu.NewCommand(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("build command: %w", err)
	}

	return cmd, nil
}
