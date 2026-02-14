// SPDX-FileCopyrightText: 2026 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package initramfs

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/aibor/cpio"
)

// WriteToTempFile writes the [fs.FS] as CPIO archive into a new temporary
// file with the given prefix and returns the absolute path to this file.
//
// If the given dir name is not empty, the file is created in this directory.
// Otherwise the default tempdir is used. See [os.CreateTemp].
func WriteToTempFile(fsys fs.FS, dir string, prefix string) (string, error) {
	file, err := os.CreateTemp(dir, prefix)
	if err != nil {
		return "", fmt.Errorf("create archive file: %w", err)
	}
	defer file.Close()

	writer := cpio.NewWriter(file)
	defer writer.Close()

	err = writer.AddFS(fsys)
	if err != nil {
		_ = os.Remove(file.Name())
		return "", fmt.Errorf("write archive: %w", err)
	}

	return file.Name(), nil
}
