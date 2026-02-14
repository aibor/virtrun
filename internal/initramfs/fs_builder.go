// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package initramfs

import (
	"errors"
	"io/fs"

	"github.com/aibor/virtrun/internal/virtfs"
)

type fileOpenFunc = virtfs.FileOpenFunc

type entry interface {
	addTo(builder builder) error
}

type builder interface {
	Add(name string, openFn fileOpenFunc) error
	Symlink(oldname, newname string) error
	MkdirAll(name string) error
}

var _ entry = directory("")

type directory string

func (d directory) addTo(builder builder) error {
	return builder.MkdirAll(string(d))
}

var _ entry = file{}

type file struct {
	Path   string
	OpenFn fileOpenFunc
}

func (v file) addTo(builder builder) error {
	return builder.Add(v.Path, v.OpenFn)
}

type copyFile struct {
	Dest   string
	Source string
	Fsys   fs.FS
}

func (f copyFile) addTo(builder builder) error {
	return builder.Add(f.Dest, func() (fs.File, error) {
		return f.Fsys.Open(f.Source)
	})
}

var _ entry = symlink{}

type symlink struct {
	Target   string
	Path     string
	MayExist bool
}

func (s symlink) addTo(builder builder) error {
	err := builder.Symlink(s.Target, s.Path)
	if err != nil && (!s.MayExist || !errors.Is(err, virtfs.ErrFileExist)) {
		return err
	}

	return nil
}
