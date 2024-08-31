// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sys

import (
	"debug/elf"
	"errors"
	"fmt"
)

var (
	ErrOSABINotSupported   = errors.New("OSABI not supported")
	ErrMachineNotSupported = errors.New("machine not supported")
)

// ValidateELF validates that ELF attributes match the requested architecture.
func ValidateELF(hdr elf.FileHeader, arch Arch) error {
	switch hdr.OSABI {
	case elf.ELFOSABI_NONE, elf.ELFOSABI_LINUX:
		// supported, pass
	default:
		return fmt.Errorf("%w: %s", ErrOSABINotSupported, hdr.OSABI)
	}

	var archReq Arch

	switch hdr.Machine {
	case elf.EM_X86_64:
		archReq = AMD64
	case elf.EM_AARCH64:
		archReq = ARM64
	case elf.EM_RISCV:
		archReq = RISCV64
	default:
		return fmt.Errorf("%w: %s", ErrMachineNotSupported, hdr.Machine)
	}

	if archReq != arch {
		return fmt.Errorf(
			"%w: %s on %s",
			ErrMachineNotSupported,
			hdr.Machine,
			arch,
		)
	}

	return nil
}
