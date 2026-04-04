// SPDX-FileCopyrightText: 2026 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package virtfs

import "errors"

// EntryFunc is a function that creates an entry to the given [FS]. It can be
// used to define a full fs without the need to handle the errors immediately.
type EntryFunc func(vfs *FS) error

// MkdirAll creates an [EntryFunc] for [FS.MkdirAll].
func MkdirAll(name string) EntryFunc {
	return func(vfs *FS) error {
		return vfs.MkdirAll(name)
	}
}

// Write creates an [EntryFunc] for [FS.Write].
func Write(name string, data []byte) EntryFunc {
	return func(vfs *FS) error {
		return vfs.Write(name, data)
	}
}

// Copy creates an [EntryFunc] for [FS.Copy].
func Copy(name string, openFn FileOpenFunc) EntryFunc {
	return func(vfs *FS) error {
		return vfs.Copy(name, openFn)
	}
}

// Symlink creates an [EntryFunc] for [FS.Symlink].
func Symlink(target, name string) EntryFunc {
	return func(vfs *FS) error {
		return vfs.Symlink(target, name)
	}
}

// MayExist creates an [EntryFunc] that wraps any other [EntryFunc] and allows
// that the file creation fails with [ErrFileExist].
func MayExist(entry EntryFunc) EntryFunc {
	return func(vfs *FS) error {
		err := entry(vfs)
		if errors.Is(err, ErrFileExist) {
			return nil
		}

		return err
	}
}
