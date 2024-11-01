// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
)

// ParseArgsError wraps errors that occur during argument parsing.
type ParseArgsError struct {
	err error
	msg string
}

func (e *ParseArgsError) Error() string {
	if e.err == nil {
		return e.msg
	}

	return fmt.Sprintf("%s: %v", e.msg, e.err)
}

func (e *ParseArgsError) Is(other error) bool {
	_, ok := other.(*ParseArgsError)
	return ok
}

func (e *ParseArgsError) Unwrap() error {
	return e.err
}
