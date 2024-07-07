// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"

	"github.com/aibor/virtrun/internal/qemu"
)

func newCommand(args qemuArgs, initramfsPath string) (*qemu.Command, error) {
	cmd := &qemu.Command{
		Executable:    args.qemu,
		Kernel:        string(args.kernel),
		Initramfs:     initramfsPath,
		Machine:       args.machine,
		CPU:           args.cpu,
		Memory:        args.memory.value,
		SMP:           args.smp.value,
		TransportType: args.transport.TransportType,
		NoKVM:         args.noKVM,
		Verbose:       args.verbose,
	}

	// In order to be useful with "go test -exec", rewrite the file based flags
	// so the output can be passed from guest to kernel via consoles.
	if !args.noGoTestFlagRewrite {
		cmd.ProcessGoTestFlags()
	}

	// Do some simple input validation to catch most obvious issues.
	err := cmd.Validate()
	if err != nil {
		return nil, fmt.Errorf("validate qemu command: %v", err)
	}

	return cmd, nil
}
