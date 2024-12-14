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

// Special file system types.
const (
	FSTypeBpf      FSType = "bpf"
	FSTypeCgroup2  FSType = "cgroup2"
	FSTypeConfig   FSType = "configfs"
	FSTypeDebug    FSType = "debugfs"
	FSTypeDevPts   FSType = "devpts"
	FSTypeDevTmp   FSType = "devtmpfs"
	FSTypeFuseCtl  FSType = "fusectl"
	FSTypeHugeTlb  FSType = "hugetlbfs"
	FSTypeMqueue   FSType = "mqueue"
	FSTypeProc     FSType = "proc"
	FSTypePstore   FSType = "pstore"
	FSTypeSecurity FSType = "securityfs"
	FSTypeSys      FSType = "sysfs"
	FSTypeTmp      FSType = "tmpfs"
	FSTypeTracing  FSType = "tracefs"

	defaultDirMode = 0o755
)

// MountOptions contains parameters for a mount point.
type MountOptions struct {
	// FSType is the files system type. It must be set to an available [FSType].
	FSType FSType

	// Source is the source device to mount. Can be empty for all the special
	// file system types [FSType]s. If empty it is set to the string of the
	// type.
	Source string

	// Flags are optional mount flags as defined by mount(2).
	Flags MountFlags

	// Data are optional additional parameters that depend of the [FSType] used.
	Data string

	// MayFail determines if the mount operation may fail. If set to true, a
	// mount error does not fail a [MountAll] operation. Instead, a warning is
	// printed to stdout and the next mount point is tried.
	MayFail bool
}

// Mount mounts the system file system of [FSType] at the given path.
//
// If path does not exist, it is created. An error is returned if this or the
// mount syscall fails.
func Mount(path string, opts MountOptions) error {
	err := os.MkdirAll(path, defaultDirMode)
	if err != nil {
		return fmt.Errorf("mkdir %s: %w", path, err)
	}

	return mount(path, opts.Source, string(opts.FSType), opts.Flags, opts.Data)
}

// MountPoints is a collection of MountPoints.
type MountPoints map[string]MountOptions

// MountAll mounts the given set of system file systems.
//
// The mounts are executed in lexicographic order of the paths.
func MountAll(mountPoints MountPoints) error {
	sortedPaths := slices.Sorted(maps.Keys(mountPoints))
	for _, path := range sortedPaths {
		opts := mountPoints[path]
		if err := Mount(path, opts); err != nil {
			if !opts.MayFail {
				return err
			}

			PrintWarning(err)
		}
	}

	return nil
}

// Symlinks is a collection of symbolic links. Keys are symbolic links to
// create with the value being the target to link to.
type Symlinks map[string]string

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
