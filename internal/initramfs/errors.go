// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package initramfs

import (
	"errors"
)

var (
	// ErrTreeNodeNotDir is returned if a tree node is supposed to be a directory
	// but is not.
	ErrTreeNodeNotDir = errors.New("tree node is not a directory")

	// ErrTreeNodeNotExists is returned if a tree node that is looked up does not
	// exist.
	ErrTreeNodeNotExists = errors.New("tree node does not exist")

	// ErrTreeNodeExists is returned if a tree node exists that was not expected.
	ErrTreeNodeExists = errors.New("tree node already exists")

	// ErrTreeNodeTypeUnknown is returned if the [TreeNodeType] is unknown.
	ErrTreeNodeTypeUnknown = errors.New("unknown tree node type")

	// ErrNotRegularFile is returned if the source is not a regular file.
	ErrNotRegularFile = errors.New("source is not a regular file")

	// ErrNoInterpreter is returned if no interpreter is found in an ELF file.
	ErrNoInterpreter = errors.New("no interpreter in ELF file")

	// ErrNotELFFile is returned if the file does not have an ELF magic number.
	ErrNotELFFile = errors.New("is not an ELF file")
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
