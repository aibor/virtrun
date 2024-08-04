// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package internal

import (
	"fmt"

	"github.com/aibor/virtrun/internal/qemu"
	"github.com/aibor/virtrun/sysinit"
)

type QemuArgs struct {
	QemuBin             string
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

func NewQemuCommand(args QemuArgs, initramfsPath string) (*qemu.Command, error) {
	cmd := &qemu.Command{
		Executable:    args.QemuBin,
		Kernel:        string(args.Kernel),
		Initramfs:     initramfsPath,
		Machine:       args.Machine,
		CPU:           args.CPU,
		Memory:        args.Memory.Value,
		SMP:           args.SMP.Value,
		TransportType: args.TransportType,
		InitArgs:      args.InitArgs,
		ExtraArgs:     args.ExtraArgs,
		NoKVM:         args.NoKVM,
		Verbose:       args.Verbose,
		ExitCodeFmt:   sysinit.ExitCodeFmt,
	}

	// In order to be useful with "go test -exec", rewrite the file based flags
	// so the output can be passed from guest to kernel via consoles.
	if !args.NoGoTestFlagRewrite {
		cmd.ProcessGoTestFlags()
	}

	// Do some simple input validation to catch most obvious issues.
	err := cmd.Validate()
	if err != nil {
		return nil, fmt.Errorf("validate qemu command: %w", err)
	}

	return cmd, nil
}
