// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package qemu

import (
	"errors"
	"fmt"
)

var (
	// ErrGuestNoExitCodeFound is returned if no exit code matching the
	// [Command.ExitCodeFmt] is printed by the guest and no other error is
	// found.
	ErrGuestNoExitCodeFound = errors.New("init did not print exit code")

	// ErrGuestPanic is returned if a kernel panic occurred in the guest
	// system.
	ErrGuestPanic = errors.New("system panicked")

	// ErrGuestOom is returned if the guest system ran out of memory.
	ErrGuestOom = errors.New("system ran out of memory")

	// ErrGuestNonZeroExitCode is returned if the guest did not return exit
	// code 0.
	ErrGuestNonZeroExitCode = errors.New("exit code not 0")

	// ErrTransportTypeInvalid is returned if a transport type is invalid.
	ErrTransportTypeInvalid = errors.New("unknown transport type")

	// ErrArgumentCollision is returned if two [Argument]s are considered equal.
	ErrArgumentCollision = errors.New("colliding args")

	// ErrConsoleNoOutput is returned if a console did not output anything. It
	// might be caused by wrong [TransportType], missing mount of /dev in the
	// guest system or wrong path the guest process writes to.
	ErrConsoleNoOutput = errors.New("console did not output anything")
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
func (*ArgumentError) Is(other error) bool {
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

	return scope + ": " + e.Err.Error()
}

// Is implements the [errors.Is] interface.
func (*CommandError) Is(other error) bool {
	_, ok := other.(*CommandError)
	return ok
}

// Unwrap implements the [errors.Unwrap] interface.
func (e *CommandError) Unwrap() error {
	return e.Err
}

// ConsoleError wraps any error occurring during console output processing.
type ConsoleError struct {
	Name string
	Err  error
}

// Error implements the [error] interface.
func (e *ConsoleError) Error() string {
	return fmt.Sprintf("console %s: %v", e.Name, e.Err.Error())
}

// Is implements the [errors.Is] interface.
func (*ConsoleError) Is(other error) bool {
	_, ok := other.(*ConsoleError)
	return ok
}

// Unwrap implements the [errors.Unwrap] interface.
func (e *ConsoleError) Unwrap() error {
	return e.Err
}
