// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package virtrun

import (
	"context"
	"fmt"
	"io"

	"github.com/aibor/virtrun/internal/qemu"
	"github.com/aibor/virtrun/internal/sys"
)

const (
	cpuDefault = "max"

	memMin     = 128
	memMax     = 16384
	memDefault = 256

	smpMin     = 1
	smpMax     = 16
	smpDefault = 1
)

// Spec describes a single [Run].
//
// It is split into parameters required for the [qemu.CommandSpec] and
// parameters required for building the initramfs archive file.
type Spec struct {
	Qemu      Qemu
	Initramfs Initramfs
}

// NewSpec creates a new [Spec] with defaults set for the given architecture.
func NewSpec(arch sys.Arch) (*Spec, error) {
	var (
		qemuExecutable    string
		qemuMachine       string
		qemuTransportType qemu.TransportType
	)

	switch arch {
	case sys.AMD64:
		qemuExecutable = "qemu-system-x86_64"
		qemuMachine = "q35"
		qemuTransportType = qemu.TransportTypePCI
	case sys.ARM64:
		qemuExecutable = "qemu-system-aarch64"
		qemuMachine = "virt"
		qemuTransportType = qemu.TransportTypeMMIO
	case sys.RISCV64:
		qemuExecutable = "qemu-system-riscv64"
		qemuMachine = "virt"
		qemuTransportType = qemu.TransportTypeMMIO
	default:
		return nil, sys.ErrArchNotSupported
	}

	args := &Spec{
		Qemu: Qemu{
			Executable:    qemuExecutable,
			Machine:       qemuMachine,
			TransportType: qemuTransportType,
			CPU:           cpuDefault,
			Memory: LimitedUintFlag{
				memDefault,
				memMin,
				memMax,
				"MB",
			},
			SMP: LimitedUintFlag{
				smpDefault,
				smpMin,
				smpMax,
				"",
			},
			NoKVM: !arch.KVMAvailable(),
			ExtraArgs: []qemu.Argument{
				qemu.UniqueArg("display", "none"),
				qemu.UniqueArg("monitor", "none"),
				qemu.UniqueArg("no-reboot", ""),
				qemu.UniqueArg("nodefaults", ""),
				qemu.UniqueArg("no-user-config", ""),
			},
		},
	}

	return args, nil
}

// Run runs with the given [Spec].
//
// An initramfs archive file is built and used for running QEMU. It returns no
// error if the run succeeds. To succeed, the guest system must explicitly
// communicate exit code 0. The built initramfs archive file is removed, unless
// [Spec.Initramfs.Keep] is set to true.
func Run(
	ctx context.Context,
	spec *Spec,
	outWriter,
	errWriter io.Writer,
) error {
	err := Validate(spec)
	if err != nil {
		return fmt.Errorf("validate: %w", err)
	}

	path, removeFn, err := BuildInitramfsArchive(ctx, spec.Initramfs)
	if err != nil {
		return err
	}
	defer removeFn() //nolint:errcheck

	cmd, err := NewQemuCommand(ctx, spec.Qemu, path)
	if err != nil {
		return err
	}

	err = cmd.Run(outWriter, errWriter)
	if err != nil {
		return fmt.Errorf("qemu run: %w", err)
	}

	return nil
}
