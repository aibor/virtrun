// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package initramfs

import (
	"fmt"
	"io"
	"io/fs"

	"github.com/cavaliergopher/cpio"
)

const numLinks = 2

var _ FileWriter = (*CPIOFileWriter)(nil)

// CPIOFileWriter extends [cpio.Writer] to implement [FileWriter].
type CPIOFileWriter struct {
	*cpio.Writer
}

// NewCPIOFileWriter creates a new archive writer.
func NewCPIOFileWriter(w io.Writer) *CPIOFileWriter {
	return &CPIOFileWriter{cpio.NewWriter(w)}
}

// WriteFile writes the given [fs.File] to the archive with the given path.
//
// File details are read from the file's [fs.FileInfo].
func (w *CPIOFileWriter) WriteFile(path string, file fs.File) error {
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}

	header, err := cpio.FileInfoHeader(info, "")
	if err != nil {
		return fmt.Errorf("header from file: %w", err)
	}

	// Override name from source with passed name for the file in the archive.
	header.Name = path
	if info.IsDir() {
		// Fix number of links for directories.
		header.Links = numLinks
	}

	err = w.WriteHeader(header)
	if err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	// Directories do not have a body and fail on [fs.File.Read].
	if info.IsDir() {
		return nil
	}

	_, err = io.Copy(w, file)
	if err != nil {
		return fmt.Errorf("write body: %w", err)
	}

	return nil
}
