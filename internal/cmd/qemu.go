// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"

	"github.com/aibor/virtrun/internal/qemu"
)

type QemuArgs struct {
	QemuBin             string
	Kernel              FilePath
	Machine             string
	CPU                 string
	SMP                 limitedUintFlag
	Memory              limitedUintFlag
	Transport           transportType
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
		TransportType: args.Transport.TransportType,
		InitArgs:      args.InitArgs,
		ExtraArgs:     args.ExtraArgs,
		NoKVM:         args.NoKVM,
		Verbose:       args.Verbose,
	}

	// In order to be useful with "go test -exec", rewrite the file based flags
	// so the output can be passed from guest to kernel via consoles.
	if !args.NoGoTestFlagRewrite {
		cmd.ProcessGoTestFlags()
	}

	// Do some simple input validation to catch most obvious issues.
	err := cmd.Validate()
	if err != nil {
		return nil, fmt.Errorf("validate qemu command: %v", err)
	}

	return cmd, nil
}
