// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package initramfs

// TreeNodeType defines the type of a [TreeNode].
type TreeNodeType int

const (
	// TreeNodeTypeRegular is a regular file. It is copied completely into the
	// archive.
	TreeNodeTypeRegular TreeNodeType = iota

	// TreeNodeTypeDirectory is a directory is created in the archive. Parent
	// directories are not created automatically. They must be created
	// beforehand.
	TreeNodeTypeDirectory

	// TreeNodeTypeLink is a symbolic link in the archive.
	TreeNodeTypeLink
)
