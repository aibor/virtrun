// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"errors"
	"flag"
	"fmt"
)

var (
	// ErrHelp aliases [flag.ErrHelp].
	ErrHelp = flag.ErrHelp

	// ErrReadBuildInfo is returned when the Go build info cannot be read.
	ErrReadBuildInfo = errors.New("can't read build info")

	// ErrNotRegularFile is returned when attempting to read a file that is not
	// a regular file.
	ErrNotRegularFile = errors.New("not a regular file")

	// ErrValueOutOfRange is returned when a given value is outside the
	// supported range.
	ErrValueOutOfRange = errors.New("value is outside of range")
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

// Is returns true if the given other error is a [ParseArgsError].
func (*ParseArgsError) Is(other error) bool {
	_, ok := other.(*ParseArgsError)
	return ok
}

func (e *ParseArgsError) Unwrap() error {
	return e.err
}
