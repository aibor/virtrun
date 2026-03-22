// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"errors"
	"fmt"
)

var (
	// ErrNotPidOne is returned if the process is expected to be run as PID 1
	// but is not.
	ErrNotPidOne = errors.New("process does not have ID 1")
	// ErrPanic is returned if a [Func] panicked.
	ErrPanic = errors.New("function panicked")
	// ErrInvalidConfig is returned in case of errors parsing the configuration.
	ErrInvalidConfig = errors.New("invalid config string")
	// ErrInvalidIfaceName is returned if a network interface name does not
	// match the kernels requirements.
	ErrInvalidIfaceName = errors.New("invalid interface name")
)

// OptionalMountError is a collection of errors that occurred for mount points
// that may fail.
type OptionalMountError []error

func (e OptionalMountError) Error() string {
	return fmt.Sprintf("optional mount errors: %q", []error(e))
}

// Is returns true if the given other error is an [OptionalMountError].
func (OptionalMountError) Is(other error) bool {
	_, ok := other.(OptionalMountError)
	return ok
}

func (e OptionalMountError) Unwrap() []error {
	return e
}
