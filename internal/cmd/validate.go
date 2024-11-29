// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"os/exec"

	"github.com/aibor/virtrun/internal/virtrun"
)

// Validate file parameters of the given [Spec].
func Validate(spec *virtrun.Spec) error {
	// Check files are actually present.
	_, err := exec.LookPath(spec.Qemu.Executable)
	if err != nil {
		return fmt.Errorf("qemu binary: %w", err)
	}

	err = ValidateFilePath(spec.Qemu.Kernel)
	if err != nil {
		return fmt.Errorf("kernel file: %w", err)
	}

	for _, file := range spec.Initramfs.Files {
		err := ValidateFilePath(file)
		if err != nil {
			return fmt.Errorf("additional file: %w", err)
		}
	}

	for _, file := range spec.Initramfs.Modules {
		err := ValidateFilePath(file)
		if err != nil {
			return fmt.Errorf("module: %w", err)
		}
	}

	err = ValidateFilePath(spec.Initramfs.Binary)
	if err != nil {
		return fmt.Errorf("main binary: %w", err)
	}

	return nil
}
