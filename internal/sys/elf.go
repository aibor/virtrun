// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package sys

import (
	"debug/elf"
	"fmt"
)

// ValidateELF validates that ELF attributes match the requested architecture.
func ValidateELF(hdr elf.FileHeader, arch Arch) error {
	switch hdr.OSABI {
	case elf.ELFOSABI_NONE, elf.ELFOSABI_LINUX:
		// supported, pass
	default:
		return fmt.Errorf("OSABI not supported: %s", hdr.OSABI)
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
		return fmt.Errorf("machine type not supported: %s", hdr.Machine)
	}

	if archReq != arch {
		return fmt.Errorf("machine %s not supported for %s", hdr.Machine, arch)
	}

	return nil
}
