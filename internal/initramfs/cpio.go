// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package initramfs

import (
	"io"
	"io/fs"
	"os"

	"github.com/cavaliergopher/cpio"
)

// CPIOFSWriter extends [cpio.Writer] by [CPIOFSWriter.AddFS] in the same way
// archive/tar and archive/zip implement it.
type CPIOFSWriter struct {
	*cpio.Writer
}

// NewCPIOFSWriter creates a new archive writer.
func NewCPIOFSWriter(w io.Writer) *CPIOFSWriter {
	return &CPIOFSWriter{cpio.NewWriter(w)}
}

// AddFS adds the files from fs.FS to the archive.
//
// It walks the directory tree starting at the root of the filesystem adding
// each file to the tar archive while maintaining the directory structure.
func (w *CPIOFSWriter) AddFS(fsys fs.FS) error {
	return fs.WalkDir(fsys, ".", func( //nolint:wrapcheck
		name string, d fs.DirEntry, err error,
	) error {
		if err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err //nolint:wrapcheck
		}

		header, err := cpio.FileInfoHeader(info, "")
		if err != nil {
			return &PathError{
				Op:   "header from file",
				Path: name,
				Err:  err,
			}
		}

		// Override name from source with passed name for the file in the
		// archive.
		header.Name = name

		err = w.WriteHeader(header)
		if err != nil {
			return &PathError{
				Op:   "write header",
				Path: name,
				Err:  err,
			}
		}

		err = w.writeBody(fsys, name, info.Mode().Type())
		if err != nil {
			return &PathError{
				Op:   "write body",
				Path: name,
				Err:  err,
			}
		}

		return nil
	})
}

func (w *CPIOFSWriter) writeBody(
	fsys fs.FS,
	name string,
	typ fs.FileMode,
) error {
	switch typ {
	case os.ModeDir:
		// Directories do not have a body and fail on [fs.File.Read].
		return nil
	case fs.ModeSymlink:
		linkName, err := ReadLink(fsys, name)
		if err != nil {
			return err
		}

		_, err = w.Write([]byte(linkName))

		return err //nolint:wrapcheck
	case 0:
		file, err := fsys.Open(name)
		if err != nil {
			return err //nolint:wrapcheck
		}
		defer file.Close()

		_, err = io.Copy(w, file)

		return err //nolint:wrapcheck
	default:
		return ErrFileInvalid
	}
}
