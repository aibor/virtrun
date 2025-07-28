// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package exitcode

import (
	"errors"
	"fmt"
)

// Error is an exit code that is considered an error.
type Error int

func (e Error) Error() string {
	return fmt.Sprintf("non-zero exit code: %d", e)
}

func (Error) Is(other error) bool {
	_, ok := other.(Error)
	return ok
}

// Code returns the exit code as basic int type.
func (e Error) Code() int {
	return int(e)
}

// From returns an exit code based on the given error and if the error was an
// [Error].
//
// If the error is nil, the exit code is 0. If the error is an [Error] the exit
// code is the return value of [Error.Code]. Otherwise the exit code is -1.
func From(err error) (int, bool) {
	if err == nil {
		return 0, false
	}

	var exitErr Error
	if errors.As(err, &exitErr) {
		return exitErr.Code(), true
	}

	return -1, false
}
