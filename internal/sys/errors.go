// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sys

import "errors"

var (
	// ErrNoInterpreter is returned if no interpreter is found in an ELF file.
	ErrNoInterpreter = errors.New("no interpreter in ELF file")

	// ErrNotELFFile is returned if the file does not have an ELF magic number.
	ErrNotELFFile = errors.New("is not an ELF file")
)
