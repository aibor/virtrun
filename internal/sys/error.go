// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sys

import (
	"errors"
	"fmt"
)

var (
	// ErrNotELFFile is returned if the file does not have an ELF magic number.
	ErrNotELFFile = errors.New("is not an ELF file")

	// ErrOSABINotSupported is returned if the OS ABI of an ELF file is not
	// supported.
	ErrOSABINotSupported = errors.New("OSABI not supported")

	// ErrMachineNotSupported is returned if the machine type of an ELF file
	// is not supported.
	ErrMachineNotSupported = errors.New("machine type not supported")

	// ErrEmptyPath is returned if an empty path is given.
	ErrEmptyPath = errors.New("path must not be empty")

	// ErrArchNotSupported is returned if the requested architecture is not
	// supported for the requested operation.
	ErrArchNotSupported = errors.New("architecture not supported")
)

// LDDExecError wraps errors that result when executing the "ldd" command.
// Along with the error the output received on stdout is added to the error
// message.
type LDDExecError struct {
	Err    error
	Stderr string
}

// Error implements the [error] interface.
func (e *LDDExecError) Error() string {
	return fmt.Sprintf("ldd execution failed: %v: %s", e.Err, e.Stderr)
}

// Is implements the [errors.Is] interface.
func (*LDDExecError) Is(other error) bool {
	_, ok := other.(*LDDExecError)
	return ok
}

// Unwrap implements the [errors.Unwrap] interface.
func (e *LDDExecError) Unwrap() error {
	return e.Err
}
