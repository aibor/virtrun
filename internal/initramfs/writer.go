// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package initramfs

import "io/fs"

// Writer defines initramfs archive writer interface.
type Writer interface {
	WriteRegular(string, fs.File, fs.FileMode) error
	WriteDirectory(string) error
	WriteLink(string, string) error
}
