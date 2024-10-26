// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package virtrun

import (
	"fmt"
	"os/exec"

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

type Virtrun struct {
	Qemu      Qemu
	Initramfs Initramfs
	Version   bool
	Debug     bool
}

func New(arch sys.Arch) (*Virtrun, error) {
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

	args := &Virtrun{
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
		Initramfs: Initramfs{
			Arch: arch,
		},
	}

	return args, nil
}

func (v *Virtrun) Validate() error {
	// Check files are actually present.
	if _, err := exec.LookPath(v.Qemu.Executable); err != nil {
		return fmt.Errorf("check qemu binary: %w", err)
	}

	if err := v.Qemu.Kernel.Check(); err != nil {
		return fmt.Errorf("check kernel file: %w", err)
	}

	for _, file := range v.Initramfs.Files {
		if err := FilePath(file).Check(); err != nil {
			return fmt.Errorf("check file: %w", err)
		}
	}

	for _, file := range v.Initramfs.Modules {
		if err := FilePath(file).Check(); err != nil {
			return fmt.Errorf("check module: %w", err)
		}
	}

	if err := v.Initramfs.Binary.CheckBinary(v.Initramfs.Arch); err != nil {
		return fmt.Errorf("check main binary: %w", err)
	}

	return nil
}
