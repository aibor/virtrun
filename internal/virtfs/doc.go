// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

// Package virtfs provides a virtual file tree implementation. It can be used to
// build a filesystem tree that can be written to a new filesystem or archive
// that takes an [io/fs.FS]. The package supports symbolic links and implements
// [io/fs.ReadLinkFS]. For memory efficiency, regular files are not copied into
// the virtual filesystem. Instead, their original source paths are mapped to
// virtual filesystem paths. Opening a virtual file opens the original file
// underneath.
package virtfs
