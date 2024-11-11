// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package virtrun

import (
	"debug/elf"
	"fmt"
	"io"
	"os/exec"

	"github.com/aibor/virtrun/internal/sys"
)

// Validate file parameters of the given [Spec].
func Validate(spec *Spec) error {
	// Check files are actually present.
	_, err := exec.LookPath(spec.Qemu.Executable)
	if err != nil {
		return fmt.Errorf("qemu binary: %w", err)
	}

	err = spec.Qemu.Kernel.Validate()
	if err != nil {
		return fmt.Errorf("kernel file: %w", err)
	}

	for _, file := range spec.Initramfs.Files {
		err := (*FilePath)(&file).Validate()
		if err != nil {
			return fmt.Errorf("additional file: %w", err)
		}
	}

	for _, file := range spec.Initramfs.Modules {
		err := (*FilePath)(&file).Validate()
		if err != nil {
			return fmt.Errorf("module: %w", err)
		}
	}

	err = spec.Initramfs.Binary.ValidateBinary(spec.Initramfs.Arch)
	if err != nil {
		return fmt.Errorf("main binary: %w", err)
	}

	return nil
}

// ValidateELF validates that ELF attributes match the requested architecture.
func ValidateELF(file io.ReaderAt, arch sys.Arch) error {
	elfFile, err := elf.NewFile(file)
	if err != nil {
		return fmt.Errorf("new: %w", err)
	}

	switch elfFile.OSABI {
	case elf.ELFOSABI_NONE, elf.ELFOSABI_LINUX:
		// supported, pass
	default:
		return fmt.Errorf("%w: %s", ErrOSABINotSupported, elfFile.OSABI)
	}

	var archReq sys.Arch

	//nolint:exhaustive
	switch elfFile.Machine {
	case elf.EM_X86_64:
		archReq = sys.AMD64
	case elf.EM_AARCH64:
		archReq = sys.ARM64
	case elf.EM_RISCV:
		archReq = sys.RISCV64
	}

	if archReq != arch {
		return fmt.Errorf(
			"%w: %s on %s",
			ErrMachineNotSupported,
			elfFile.Machine,
			arch,
		)
	}

	return nil
}
