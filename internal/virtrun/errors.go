// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package virtrun

import "errors"

var (
	ErrEmptyFilePath  = errors.New("file path must not be empty")
	ErrNotRegularFile = errors.New("not a regular file")

	// ErrOSABINotSupported is returned if the OS ABI of an ELF file is not
	// supported.
	ErrOSABINotSupported = errors.New("OSABI not supported")

	// ErrMachineNotSupported is returned if the machine type of an ELF file
	// is not supported.
	ErrMachineNotSupported = errors.New("machine type not supported")
)
