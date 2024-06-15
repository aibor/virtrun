// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package initramfs

import "io/fs"

// Writer defines initramfs archive writer interface.
type Writer interface {
	WriteRegular(path string, source fs.File, mode fs.FileMode) error
	WriteDirectory(path string) error
	WriteLink(path string, target string) error
}
