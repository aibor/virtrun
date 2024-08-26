// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package initramfs

import (
	"errors"
)

var (
	// ErrNodeNotDir is returned if a tree node is supposed to be a directory
	// but is not.
	ErrNodeNotDir = errors.New("tree node is not a directory")

	// ErrNodeNotExists is returned if a tree node that is looked up does not exist.
	ErrNodeNotExists = errors.New("tree node does not exist")

	// ErrNodeExists is returned if a tree node exists that was not expected.
	ErrNodeExists = errors.New("tree node already exists")

	// ErrNotRegularFile is returned if the source is not a regular file.
	ErrNotRegularFile = errors.New("source is not a regular file")

	// ErrFileTypeUnknown is returned if the file type is unknown.
	ErrFileTypeUnknown = errors.New("unknown file type")
)

// ArchiveError is returned if there is an error writing the archive.
type ArchiveError struct {
	Op   string
	Path string
	Err  error
}

// Error implements the [error] interface.
func (e *ArchiveError) Error() string {
	return "archive " + e.Op + " " + e.Path + ": " + e.Err.Error()
}

// Is implements the [errors.Is] interface.
func (e *ArchiveError) Is(other error) bool {
	err, ok := other.(*ArchiveError)
	return ok && e.Op == err.Op
}

// Unwrap implements the [errors.Unwrap] interface.
func (e *ArchiveError) Unwrap() error {
	return e.Err
}
