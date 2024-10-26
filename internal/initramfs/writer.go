// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package initramfs

import (
	"io/fs"
)

// FileWriter defines initramfs archive writer interface.
type FileWriter interface {
	WriteFile(path string, file fs.File) error
}

// WriteFS writes the given [fs.FS] to the given [FileWriter].
func WriteFS(fsys fs.FS, writer FileWriter) error {
	fn := func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		file, err := fsys.Open(path)
		if err != nil {
			return err //nolint:wrapcheck
		}
		defer file.Close()

		err = writer.WriteFile(path, file)
		if err != nil {
			return &PathError{
				Op:   "archive write",
				Path: path,
				Err:  err,
			}
		}

		return nil
	}

	return fs.WalkDir(fsys, ".", fn) //nolint:wrapcheck
}
