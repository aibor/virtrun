// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package initramfs

import (
	"fmt"
	"io"
	"io/fs"

	"github.com/cavaliergopher/cpio"
)

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
	return w.cpioWriter.Close()
}

// Flush writes the data to the underlying [io.Writer].
func (w *CPIOWriter) Flush() error {
	return w.cpioWriter.Flush()
}

// writeHeader writes the cpio header.
func (w *CPIOWriter) writeHeader(hdr *cpio.Header) error {
	if err := w.cpioWriter.WriteHeader(hdr); err != nil {
		return fmt.Errorf("write header for %s: %v", hdr.Name, err)
	}
	return nil
}

// WriteDirectory add a directory entry for the given path to the archive.
func (w *CPIOWriter) WriteDirectory(path string) error {
	header := &cpio.Header{
		Name:  path,
		Mode:  cpio.TypeDir | cpio.ModePerm,
		Links: 2,
	}
	return w.writeHeader(header)
}

// WriteLink adds a symbolic link for the given path pointing to the given
// target.
func (w *CPIOWriter) WriteLink(path, target string) error {
	header := &cpio.Header{
		Name: path,
		Mode: cpio.TypeSymlink | cpio.ModePerm,
		Size: int64(len(target)),
	}
	if err := w.writeHeader(header); err != nil {
		return err
	}

	// Body of a link is the path of the target file.
	if _, err := w.cpioWriter.Write([]byte(target)); err != nil {
		return fmt.Errorf("write body for %s: %v", path, err)
	}

	return nil
}

// WriteRegular copies the exisiting file from source into the archive.
func (w *CPIOWriter) WriteRegular(path string, source fs.File, mode fs.FileMode) error {
	info, err := source.Stat()
	if err != nil {
		return fmt.Errorf("read info: %v", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("not a regular file: %s", source)
	}

	cpioHdr, err := cpio.FileInfoHeader(info, "")
	if err != nil {
		return fmt.Errorf("create header: %v", err)
	}

	cpioHdr.Name = path
	if mode != 0 {
		cpioHdr.Mode = cpio.FileMode(mode)
	}

	if err := w.writeHeader(cpioHdr); err != nil {
		return err
	}

	if _, err := io.Copy(w.cpioWriter, source); err != nil {
		return fmt.Errorf("write body for %s: %v", path, err)
	}

	return nil
}
