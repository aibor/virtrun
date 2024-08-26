// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package qemu

import "errors"

var (
	// ErrGuestNoExitCodeFound is returned if no exit code matching the
	// [Command.ExitCodeFmt] is printed by the guest and no other error is
	// found.
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

	// ErrArgumentCollision is returned if two [Argument]s are considered equal.
	ErrArgumentCollision = errors.New("colliding args")
)

// ArgumentError indicates an issue with an input argument.
type ArgumentError struct {
	msg string
}

// Error implements the [error] interface.
func (e *ArgumentError) Error() string {
	return "argument error: " + e.msg
}

// Is implements the [errors.Is] interface.
func (e *ArgumentError) Is(other error) bool {
	_, ok := other.(*ArgumentError)
	return ok
}

// CommandError wraps any error occurred during Command execution.
type CommandError struct {
	Err      error
	Guest    bool
	ExitCode int
}

// Error implements the [error] interface.
func (e *CommandError) Error() string {
	scope := "host"
	if e.Guest {
		scope = "guest"
	}

	return "qemu " + scope + ": " + e.Err.Error()
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
