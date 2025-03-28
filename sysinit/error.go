// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"errors"
	"fmt"
)

// ErrNotPidOne may be returned if the process is expected to be run as PID 1
// but is not.
var ErrNotPidOne = errors.New("process does not have ID 1")

// ExitError is an exit code that is considered an error.
type ExitError int

func (e ExitError) Error() string {
	return fmt.Sprintf("non-zero exit code: %d", e)
}

func (ExitError) Is(other error) bool {
	_, ok := other.(ExitError)
	return ok
}

// Code returns the exit code as basic int type.
func (e ExitError) Code() int {
	return int(e)
}

// OptionalMountError is a collection of errors that occurred for mount points
// that may fail.
type OptionalMountError []error

func (e OptionalMountError) Error() string {
	return fmt.Sprintf("optional mount errors: %q", []error(e))
}

func (OptionalMountError) Is(other error) bool {
	_, ok := other.(OptionalMountError)
	return ok
}

func (e OptionalMountError) Unwrap() []error {
	return e
}
