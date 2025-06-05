// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

// ExitCodeID is the identifier string for communicating an exit code via
// stdout.
//
// matched correctly.
const ExitCodeID = ExitCodeIdentifier("SYSINIT_EXIT_CODE")

// ExitCodeIdentifier is an identifier string for communicating an exit code
// via stdout.
//
// The same instance must be used by the init program and the consumer of its
// output.
type ExitCodeIdentifier string

// Sprint prints the exit code line with the given exit code.
//
// Its return value should be written to stdout  by the init program, e.g.
// by [ExitCodeIdentifier.PrintFrom].
func (e ExitCodeIdentifier) Sprint(exitCode int) string {
	return fmt.Sprintf(e.format(), exitCode)
}

// ParseExitCode parses the given string for the exit code.
//
// The identifier can be anywhere in the string. It does not need to be at the
// beginning. Returns the exit code and whether it was found in the given
// string.
func (e ExitCodeIdentifier) ParseExitCode(str string) (int, bool) {
	start := strings.Index(str, string(e))
	if start < 0 {
		return 0, false
	}

	format := e.format()

	var exitCode int

	if _, err := fmt.Sscanf(str[start:], format, &exitCode); err != nil {
		return 0, false
	}

	return exitCode, true
}

// Printer returns an [ExitHandler] that prints an exit code for a given error.
//
// See [ExitCodeFrom] for the resulting exit codes.
func (e ExitCodeIdentifier) Printer(w io.Writer) ExitHandler {
	return func(err error) {
		exitCode := ExitCodeFrom(err)
		_, _ = fmt.Fprintln(w, e.Sprint(exitCode))
	}
}

func (e ExitCodeIdentifier) format() string {
	return string(e) + ": %d"
}

// ExitCodeFrom returns an exit code based on the given error.
//
// If the error is nil, the exit code is 0. If the error is an [ExitError]
// the exit code is the return value of [ExitError.Code]. Otherwise the exit
// code is -1.
func ExitCodeFrom(err error) int {
	if err == nil {
		return 0
	}

	var exitErr ExitError
	if errors.As(err, &exitErr) {
		return exitErr.Code()
	}

	return -1
}
