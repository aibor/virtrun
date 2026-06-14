// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package virtfs

import (
	"errors"
	"io/fs"
)

var (
	// ErrFileNotExist is returned when a tree node that is looked up does not
	// exist.
	ErrFileNotExist = fs.ErrNotExist

	// ErrFileExist is returned when a tree node exists unexpectedly.
	ErrFileExist = fs.ErrExist

	// ErrFileInvalid is returned when a file is invalid for the requested
	// operation.
	ErrFileInvalid = fs.ErrInvalid

	// ErrFileNotDir is returned when a file exists but is not a directory.
	ErrFileNotDir = errors.New("not a directory")

	// ErrFileNotRegular is returned when the source is not a regular file.
	ErrFileNotRegular = errors.New("source is not a regular file")

	// ErrInvalidArgument is returned when an invalid argument is given.
	ErrInvalidArgument = errors.New("invalid argument")

	// ErrSymlinkTooDeep is returned when there are too many symbolic links to
	// follow.
	ErrSymlinkTooDeep = errors.New("nested links too deep")
)

// PathError records an error and the operation and file path that caused it.
type PathError = fs.PathError
