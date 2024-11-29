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

	// ErrReadBuildInfo is returned if the go build info can not be read.
	ErrReadBuildInfo = errors.New("can't read build info")

	// ErrEmptyFilePath is returned if an empty file path is given.
	ErrEmptyFilePath = errors.New("file path must not be empty")

	// ErrNotRegularFile is returned if a file should be read but is not a
	// regular file.
	ErrNotRegularFile = errors.New("not a regular file")
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
