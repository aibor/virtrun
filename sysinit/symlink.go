// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"fmt"
	"os"
)

// DevSymlinks returns a map with well-known symlinks for /dev.
func DevSymlinks() Symlinks {
	return Symlinks{
		"/dev/core":   "/proc/kcore",
		"/dev/fd":     "/proc/self/fd/",
		"/dev/rtc":    "rtc0",
		"/dev/stdin":  "/proc/self/fd/0",
		"/dev/stdout": "/proc/self/fd/1",
		"/dev/stderr": "/proc/self/fd/2",
	}
}

// Symlinks is a collection of symbolic links. Keys are symbolic links to
// create with the value being the target to link to.
type Symlinks map[string]string

// CreateSymlinks creates common symbolic links in the file system.
//
// This must be run after all file systems have been mounted.
func CreateSymlinks(symlinks Symlinks) error {
	for link, target := range sortedMap(symlinks) {
		if err := os.Symlink(target, link); err != nil {
			return fmt.Errorf("create common symlink %s: %w", link, err)
		}
	}

	return nil
}

// WithSymlinks returns a setup [Func] that wraps [CreateSymlinks] and can be
// used with [Run].
func WithSymlinks(symlinks Symlinks) Func {
	return func(_ *State) error {
		return CreateSymlinks(symlinks)
	}
}
