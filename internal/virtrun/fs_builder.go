// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package virtrun

import (
	"errors"
	"io/fs"

	"github.com/aibor/virtrun/internal/virtfs"
)

type fileOpenFunc = virtfs.FileOpenFunc

type fsEntry interface {
	addTo(builder fsBuilder) error
}

type fsBuilder interface {
	Add(name string, openFn fileOpenFunc) error
	Symlink(oldname, newname string) error
	MkdirAll(name string) error
}

var _ fsEntry = directory("")

type directory string

func (d directory) addTo(builder fsBuilder) error {
	return builder.MkdirAll(string(d))
}

var _ fsEntry = file{}

type file struct {
	Path   string
	OpenFn fileOpenFunc
}

func (v file) addTo(builder fsBuilder) error {
	return builder.Add(v.Path, v.OpenFn)
}

type copyFile struct {
	Dest   string
	Source string
	Fsys   fs.FS
}

func (f copyFile) addTo(builder fsBuilder) error {
	return builder.Add(f.Dest, func() (fs.File, error) {
		return f.Fsys.Open(f.Source)
	})
}

var _ fsEntry = symlink{}

type symlink struct {
	Target   string
	Path     string
	MayExist bool
}

func (s symlink) addTo(builder fsBuilder) error {
	err := builder.Symlink(s.Target, s.Path)
	if err != nil && (!s.MayExist || !errors.Is(err, virtfs.ErrFileExist)) {
		return err
	}

	return nil
}
