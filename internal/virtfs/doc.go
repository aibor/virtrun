// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

// Package virtfs provides a virtual file tree. It is intended to be used to
// build a tree that can be written to a new file system or archive
// that takes an [io/fs.FS]. It supports symbolic links and implements
// [io/fs.ReadLinkFS].
//
// For memory efficiency regular files are not copied into the virtual fs
// itself. Instead, their original source path is just mapped to the virtual fs
// path. Opening the virtual file opens the original file underneath.
package virtfs
