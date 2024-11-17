// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package initramfs

import "io/fs"

// ReadLinkFS is a [fs.FS] with an additional method for reading the target of
// a symbolic link.
//
// Replace with [fs.ReadLinkFS] once available. See
// https://github.com/golang/go/issues/49580
type ReadLinkFS interface {
	fs.FS

	// ReadLink returns the destination a symbolic link points to. Returns an
	// error if the file at name is not a symbolic link or cannot be read.
	ReadLink(name string) (string, error)
}

type readLinkFS struct {
	fs.FS
	readLinkFn ReadLinkFunc
}

// ReadLink implements [ReadLinkFS].
func (fsys *readLinkFS) ReadLink(name string) (string, error) {
	return fsys.readLinkFn(name)
}

// ReadLinkFunc returns the destination of a symbolic link or an error in case
// the file at name is not a symbolic link or cannot be read.
type ReadLinkFunc func(name string) (string, error)

// WithReadLinkFunc extends the given [fs.FS] with the given [ReadLinkFunc]
// into a new [ReadLinkFS].
//
//nolint:ireturn
func WithReadLinkFunc(fsys fs.FS, readLinkFn ReadLinkFunc) ReadLinkFS {
	return &readLinkFS{
		FS:         fsys,
		readLinkFn: readLinkFn,
	}
}

// WithReadLinkNoFollowOpen extends the given [fs.FS] into a new [ReadLinkFS].
//
// The source [fs.FS]'s Open method must not follow symbolic links. It must open
// them directly so the destination can be read and returned by the ReadLink
// method.
//
//nolint:ireturn
func WithReadLinkNoFollowOpen(fsys fs.FS) ReadLinkFS {
	return WithReadLinkFunc(fsys, func(name string) (string, error) {
		file, err := fsys.Open(name)
		if err != nil {
			return "", err //nolint:wrapcheck
		}
		defer file.Close()

		info, err := file.Stat()
		if err != nil {
			return "", err //nolint:wrapcheck
		}

		if info.Mode().Type() != fs.ModeSymlink {
			return "", &PathError{
				Op:   "readlink",
				Path: name,
				Err:  ErrFileInvalid,
			}
		}

		b := make([]byte, info.Size())

		if _, err := file.Read(b); err != nil {
			return "", &PathError{
				Op:   "readlink",
				Path: name,
				Err:  err,
			}
		}

		return string(b), nil
	})
}
