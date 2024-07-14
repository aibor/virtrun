// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package internal

import (
	"errors"
	"fmt"
)

var ErrArchNotSupported = errors.New("architecture not supported")

// ParseArgsError wraps errors that occur during argument parsing.
type ParseArgsError struct {
	err error
	msg string
}

func (e *ParseArgsError) Error() string {
	return fmt.Sprintf("%s: %v", e.msg, e.err)
}

func (e *ParseArgsError) Is(other error) bool {
	_, ok := other.(*ParseArgsError)

	return ok
}

func (e *ParseArgsError) Unwrap() error {
	return e.err
}
