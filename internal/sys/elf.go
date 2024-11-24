// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sys

import (
	"debug/elf"
	"fmt"
	"strings"
)

// ReadELFArch returns the [sys.Arch] of the given ELF file.
//
// It returns an error if the ELF file is not for Linux or is for an
// unsupported architecture.
func ReadELFArch(fileName string) (Arch, error) {
	file, err := elfOpen(fileName)
	if err != nil {
		return "", err
	}
	defer file.Close()

	switch file.OSABI {
	case elf.ELFOSABI_NONE, elf.ELFOSABI_LINUX:
		// supported, pass
	default:
		return "", fmt.Errorf("%w: %s", ErrOSABINotSupported, file.OSABI)
	}

	switch file.Machine {
	case elf.EM_X86_64:
		return AMD64, nil
	case elf.EM_AARCH64:
		return ARM64, nil
	case elf.EM_RISCV:
		return RISCV64, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrMachineNotSupported, file.Machine)
	}
}

func elfOpen(name string) (*elf.File, error) {
	elfFile, err := elf.Open(name)
	if err != nil {
		if strings.Contains(err.Error(), "bad magic number") {
			err = ErrNotELFFile
		}

		return nil, fmt.Errorf("open %s: %w", name, err)
	}

	return elfFile, nil
}
