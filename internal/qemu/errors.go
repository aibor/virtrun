// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package qemu

import (
	"errors"
	"fmt"
)

var (
	// ErrGuestNoExitCodeFound is returned if no exit code matching the [RCFmt]
	// is printed by the guest and no other error is found.
	ErrGuestNoExitCodeFound = errors.New("guest did not print init exit code")

	// ErrGuestPanic is returned if a kernel panic occurred in the guest
	// system.
	ErrGuestPanic = errors.New("guest system panicked")

	// ErrGuestOom is returned if the guest system ran out of memory.
	ErrGuestOom = errors.New("guest system ran out of memory")

	// ErrGuestNonZeroExitCode is returned if the guest did not return exit
	// code 0.
	ErrGuestNonZeroExitCode = errors.New("guest did not return exit code 0")

	// ErrTransportTypeInvalid is returned if a transport type is invalid.
	ErrTransportTypeInvalid = errors.New("unknown transport type")
)

// CommandError wraps any error occurred during Command execution.
type CommandError struct {
	Err      error
	ExitCode int
}

// Error implements the [error] interface.
func (e *CommandError) Error() string {
	return fmt.Sprintf("qemu command error: %v", e.Err)
}

// Is implements the [errors.Is] interface.
func (e *CommandError) Is(other error) bool {
	_, ok := other.(*CommandError)

	return ok
}

// Unwrap implements the [errors.Unwrap] interface.
func (e *CommandError) Unwrap() error {
	return e.Err
}
