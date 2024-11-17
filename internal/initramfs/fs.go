// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package initramfs

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

const defaultFileMode = 0o755

// FileOpenFunc returns an open [fs.File] or an error if opening fails.
type FileOpenFunc func() (fs.File, error)

// FSAdder defines the interface required to add files to a FS.
type FSAdder interface {
	Add(name string, openFn FileOpenFunc) error
	Symlink(oldname, newname string) error
	Mkdir(name string) error
	MkdirAll(name string) error
}

var (
	_ fs.FS   = (*FS)(nil)
	_ FSAdder = (*FS)(nil)
)

// FS represents a simple [fs.FS] that supports directories, regular files and
// symbolic links
//
// Regular files that should be copied from another source can be added with
// [FS.Add].It supports adding symbolic links with [FS.Symlink]. Use [FS.Mkdir]
// or [FS.MkdirAll] to create any required directories beforehand.
type FS struct {
	root directory
}

// New creates a new [FS].
func New() *FS {
	return &FS{
		root: make(directory),
	}
}

// Open opens the named file.
//
// It returns a [PathError] in case of errors. It does not follow symbolic
// links and returns symbolic links directly.
func (fsys *FS) Open(name string) (fs.File, error) {
	file, err := fsys.open(name)
	if err != nil {
		return nil, &PathError{
			Op:   "open",
			Path: name,
			Err:  err,
		}
	}

	return file, nil
}

// ReadLink returns the target of the symbolic link at the given name.
//
// It prepares for [fs.ReadLinkFS] as proposed:
// https://github.com/golang/go/issues/49580
func (fsys *FS) ReadLink(name string) (string, error) {
	target, err := fsys.readlink(name)
	if err != nil {
		return "", &PathError{
			Op:   "readlink",
			Path: name,
			Err:  err,
		}
	}

	return target, nil
}

// Mkdir creates a new directory with the given name.
//
// It returns [PathError] in case of errors.
func (fsys *FS) Mkdir(name string) error {
	parentName, dirName := filepath.Split(clean(name))

	parent, err := fsys.subDir(clean(parentName))
	if err != nil {
		return &PathError{
			Op:   "mkdir",
			Path: name,
			Err:  err,
		}
	}

	err = parent.add(dirName, &directory{})
	if err != nil {
		return &PathError{
			Op:   "mkdir",
			Path: name,
			Err:  err,
		}
	}

	return nil
}

// MkdirAll creates a directory with the given name along with all necessary
// parents.
//
// It returns a [PathError] in case of errors. If the directory exists already,
// it does nothing and returns nil.
func (fsys *FS) MkdirAll(name string) error {
	cleaned := clean(name)

	file, err := fsys.find(cleaned)
	if err == nil {
		if file.mode().IsDir() {
			return nil
		}

		return &PathError{
			Op:   "mkdir",
			Path: name,
			Err:  ErrFileNotDir,
		}
	}

	parent := filepath.Dir(cleaned)
	if parent != cleaned {
		err = fsys.MkdirAll(parent)
		if err != nil {
			return err
		}
	}

	return fsys.Mkdir(name)
}

// Add creates a new regular file with the given name.
//
// File content is read from the file returned by the given [FileOpenFunc]. It
// returns a [PathError] in case of errors.
func (fsys *FS) Add(name string, openFn FileOpenFunc) error {
	if openFn == nil {
		return &PathError{
			Op:   "add",
			Path: name,
			Err:  fmt.Errorf("%w: openFunc is nil", ErrInvalidArgument),
		}
	}

	err := fsys.add(name, regularFile(openFn))
	if err != nil {
		return &PathError{
			Op:   "add",
			Path: name,
			Err:  err,
		}
	}

	return nil
}

// Symlink adds a new symbolic link that links to oldname at newname.
//
// It returns a [PathError] in case of errors.
func (fsys *FS) Symlink(oldname, newname string) error {
	file := symbolicLink(oldname)

	err := fsys.add(newname, file)
	if err != nil {
		return &PathError{
			Op:   "symlink",
			Path: newname,
			Err:  err,
		}
	}

	return nil
}

func (fsys *FS) subDir(name string) (*directory, error) {
	file, err := fsys.find(name)
	if err != nil {
		return nil, err
	}

	dir, isDir := file.(*directory)
	if !isDir {
		return nil, ErrFileNotDir
	}

	return dir, nil
}

func (fsys *FS) add(name string, file file) error {
	dirName, fileName := filepath.Split(clean(name))

	parent, err := fsys.subDir(clean(dirName))
	if err != nil {
		return err
	}

	err = parent.add(fileName, file)
	if err != nil {
		return err
	}

	return nil
}

func (fsys *FS) open(name string) (fs.File, error) {
	file, err := fsys.find(name)
	if err != nil {
		return nil, err
	}

	info := dirEntry{
		name: name,
		file: file,
	}

	return file.open(info)
}

func (fsys *FS) readlink(name string) (string, error) {
	file, err := fsys.find(name)
	if err != nil {
		return "", err
	}

	symlink, isSymlink := file.(symbolicLink)
	if !isSymlink {
		return "", ErrFileInvalid
	}

	return string(symlink), nil
}

//nolint:ireturn
func (fsys *FS) find(name string) (file, error) {
	var file file = &fsys.root

	if name == "" || name == "." {
		return file, nil
	}

	if !fs.ValidPath(name) {
		return nil, ErrFileInvalid
	}

	nodes := strings.Split(name, string(filepath.Separator))
	for _, name = range nodes {
		dir, isDir := file.(*directory)
		if !isDir {
			return nil, ErrFileNotExist
		}

		next, exists := (*dir)[name]
		if !exists {
			return nil, ErrFileNotExist
		}

		file = next
	}

	return file, nil
}

func clean(path string) string {
	path = filepath.Clean(path)
	return strings.TrimPrefix(path, string(filepath.Separator))
}
