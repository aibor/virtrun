// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
)

// FSType is a file system type.
type FSType string

// Special essential file system types.
const (
	FSTypeDevTmp  FSType = "devtmpfs"
	FSTypeProc    FSType = "proc"
	FSTypeSys     FSType = "sysfs"
	FSTypeTmp     FSType = "tmpfs"
	FSTypeBpf     FSType = "bpf"
	FSTypeTracing FSType = "tracefs"

	defaultDirMode = 0o755
)

// MountOptions is a single mount point for a virtual system FS.
type MountOptions struct {
	FSType  FSType
	MayFail bool
}

// MountPoints is a collection of MountPoints.
type MountPoints map[string]MountOptions

// Symlinks is a collection of symbolic links. Keys are symbolic links to
// create with the value being the target to link to.
type Symlinks map[string]string

// Mount mounts the system file system of [FSType] at the given path.
//
// If path does not exist, it is created. An error is returned if this or the
// mount syscall fails.
func Mount(path string, fsType FSType) error {
	err := os.MkdirAll(path, defaultDirMode)
	if err != nil {
		return fmt.Errorf("mkdir %s: %w", path, err)
	}

	return mount(path, "", string(fsType))
}

// MountAll mounts the given set of system file systems.
//
// The mounts are executed in lexicographic order of the paths.
func MountAll(mountPoints MountPoints) error {
	sortedPaths := slices.Sorted(maps.Keys(mountPoints))
	for _, path := range sortedPaths {
		opts := mountPoints[path]
		if err := Mount(path, opts.FSType); err != nil && !opts.MayFail {
			return err
		}
	}

	return nil
}

// CreateSymlinks creates common symbolic links in the file system.
//
// This must be run after all file systems have been mounted.
func CreateSymlinks(symlinks Symlinks) error {
	for link, target := range symlinks {
		if err := os.Symlink(target, link); err != nil {
			return fmt.Errorf("create common symlink %s: %w", link, err)
		}
	}

	return nil
}

// ListRegularFiles lists all regular files in the given directory and all
// sub directories.
func ListRegularFiles(dir string) ([]string, error) {
	var files []string

	walkFunc := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.Type().IsRegular() {
			files = append(files, path)
		}

		return nil
	}

	err := filepath.WalkDir(dir, walkFunc)
	if err != nil {
		return nil, fmt.Errorf("walk dir: %w", err)
	}

	return files, nil
}
