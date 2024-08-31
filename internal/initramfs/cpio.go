// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package initramfs

import (
	"bytes"
	"io"
	"io/fs"

	"github.com/cavaliergopher/cpio"
)

const numLinks = 2

// CPIOWriter implements [Writer] for [cpio.CPIOWriter].
type CPIOWriter struct {
	cpioWriter *cpio.Writer
}

// NewCPIOWriter creates a new archive writer.
func NewCPIOWriter(w io.Writer) *CPIOWriter {
	return &CPIOWriter{cpio.NewWriter(w)}
}

// Close closes the [Writer]. Flush is called by the underlying closer.
func (w *CPIOWriter) Close() error {
	err := w.cpioWriter.Close()
	if err != nil {
		return &ArchiveError{
			Op:  "close",
			Err: err,
		}
	}

	return nil
}

// Flush writes the data to the underlying [io.Writer].
func (w *CPIOWriter) Flush() error {
	err := w.cpioWriter.Flush()
	if err != nil {
		return &ArchiveError{
			Op:  "flush",
			Err: err,
		}
	}

	return nil
}

func (w *CPIOWriter) write(hdr *cpio.Header, src io.Reader) error {
	if err := w.cpioWriter.WriteHeader(hdr); err != nil {
		return &ArchiveError{
			Op:   "write header",
			Path: hdr.Name,
			Err:  err,
		}
	}

	if src == nil {
		return nil
	}

	if _, err := io.Copy(w.cpioWriter, src); err != nil {
		return &ArchiveError{
			Op:   "write body",
			Path: hdr.Name,
			Err:  err,
		}
	}

	return nil
}

// WriteDirectory add a directory entry for the given path to the archive.
func (w *CPIOWriter) WriteDirectory(path string) error {
	header := &cpio.Header{
		Name:  path,
		Mode:  cpio.TypeDir | cpio.ModePerm,
		Links: numLinks,
	}

	return w.write(header, nil)
}

// WriteLink adds a symbolic link for the given path pointing to the given
// target.
func (w *CPIOWriter) WriteLink(path, target string) error {
	header := &cpio.Header{
		Name: path,
		Mode: cpio.TypeSymlink | cpio.ModePerm,
		Size: int64(len(target)),
	}

	// Body of a link is the path of the target file.
	return w.write(header, bytes.NewBufferString(target))
}

// WriteRegular copies the exisiting file from source into the archive.
func (w *CPIOWriter) WriteRegular(
	path string,
	source fs.File,
	mode fs.FileMode,
) error {
	info, err := source.Stat()
	if err != nil {
		return &ArchiveError{
			Op:   "read info",
			Path: path,
			Err:  err,
		}
	}

	if !info.Mode().IsRegular() {
		return &ArchiveError{
			Op:   "check source",
			Path: path,
			Err:  ErrNotRegularFile,
		}
	}

	cpioHdr, err := cpio.FileInfoHeader(info, "")
	if err != nil {
		return &ArchiveError{
			Op:   "create header",
			Path: path,
			Err:  err,
		}
	}

	cpioHdr.Name = path
	if mode != 0 {
		cpioHdr.Mode = cpio.FileMode(mode)
	}

	return w.write(cpioHdr, source)
}
