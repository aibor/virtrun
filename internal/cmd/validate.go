// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/aibor/virtrun/internal/virtrun"
)

// validateFilePaths validates file parameters of the given [Spec].
func validateFilePaths(spec *virtrun.Spec) error {
	err := ValidateFilePath(spec.Qemu.Kernel)
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
