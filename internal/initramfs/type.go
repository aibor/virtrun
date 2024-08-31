// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package initramfs

// FileType defines the type of a [TreeNode].
type FileType int

const (
	// FileTypeRegular is a regular file. It is copied completely into the
	// archive.
	FileTypeRegular FileType = iota
	// FileTypeDirectory is a directory is created in the archive. Parent
	// directories are not created automatically. They must be created
	// beforehand.
	FileTypeDirectory
	// FileTypeLink is a symbolic link in the archive.
	FileTypeLink
	// FileTypeVirtual is like [FileTypeRegular] but with its content written
	// from an io.Reader instead of being copied from the fs.
	FileTypeVirtual
)
