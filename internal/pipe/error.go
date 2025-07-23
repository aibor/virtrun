// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package pipe

import (
	"errors"
	"fmt"
)

var (
	// ErrNoOutput is returned if a pipe did not output anything. It
	// might be caused by wrong [TransportType], missing mount of /dev in the
	// guest system or wrong path the guest process writes to.
	ErrNoOutput = errors.New("pipe did not output anything")

	// ErrWaitTimeout is returned if the pipe termination took too long and
	// ran out of allowed time.
	ErrWaitTimeout = errors.New("pipe wait timed out")
)

// Error wraps any error occurring during pipe processing.
type Error struct {
	Name string
	Err  error
}

// Error implements the [error] interface.
func (e *Error) Error() string {
	return fmt.Sprintf("pipe %s: %v", e.Name, e.Err.Error())
}

// Is implements the [errors.Is] interface.
func (*Error) Is(other error) bool {
	_, ok := other.(*Error)
	return ok
}

// Unwrap implements the [errors.Unwrap] interface.
func (e *Error) Unwrap() error {
	return e.Err
}
