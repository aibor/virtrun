// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package initramfs

import "io/fs"

// ReadLinkFS is a [fs.FS] with an additional method for reading the target of
// a symbolic link.
//
// Replace with [fs.ReadLinkFS] once available (planned for 1.25). See
// https://github.com/golang/go/issues/49580
type ReadLinkFS interface {
	fs.FS

	ReadLink(name string) (string, error)
	Lstat(name string) (fs.FileInfo, error)
}

// ReadLink returns the destination a symbolic links points to.
//
// The given [fs.FS], must implement [readLinkFS], otherwise ErrFileInvalid is
// returned.
func ReadLink(fsys fs.FS, name string) (string, error) {
	rlFS, ok := fsys.(ReadLinkFS)
	if !ok {
		return "", &PathError{
			Op:   "readlink",
			Path: name,
			Err:  ErrFileInvalid,
		}
	}

	return rlFS.ReadLink(name) //nolint:wrapcheck
}

type readLinkFS struct {
	fs.FS
	readLinkFn func(name string) (string, error)
	lstatFn    func(name string) (fs.FileInfo, error)
}

// ReadLink implements [ReadLinkFS].
func (fsys *readLinkFS) ReadLink(name string) (string, error) {
	return fsys.readLinkFn(name)
}

// Lstat implements [ReadLinkFS].
func (fsys *readLinkFS) Lstat(name string) (fs.FileInfo, error) {
	return fsys.lstatFn(name)
}

// WithReadLinkNoFollowOpen extends the given [fs.FS] into a new [ReadLinkFS].
//
// The source [fs.FS]'s Open method must not follow symbolic links. It must open
// them directly so the destination can be read and returned by the ReadLink
// method.
//
// This is a workaround until the standard library's implementations implement
// [ReadLinkFS] themself (planned for 1.25). See
// https://github.com/golang/go/issues/49580
func WithReadLinkNoFollowOpen(fsys fs.FS) fs.FS {
	return &readLinkFS{
		FS: fsys,
		readLinkFn: func(name string) (string, error) {
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
		},
		lstatFn: func(name string) (fs.FileInfo, error) {
			return fs.Stat(fsys, name)
		},
	}
}
