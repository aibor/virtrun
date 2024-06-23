// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package sysinit

import (
	"fmt"
	"os"
	"syscall"
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

// MountPoint is a single mount point for a virtual system FS.
type MountPoint struct {
	Path   string
	FSType FSType
}

// MountPoints is a collection of MountPoints.
type MountPoints []MountPoint

// Symlinks is a collection of symbolic links. Keys are symbolic links to
// create with the value being the target to link to.
type Symlinks map[string]string

// Mount mounts the special file system with type FsType at the given path.
//
// If path does not exist, it is created. An error is returned if this or the
// mount syscall fails.
func Mount(mount MountPoint) error {
	err := os.MkdirAll(mount.Path, defaultDirMode)
	if err != nil {
		return fmt.Errorf("mkdir %s: %v", mount.Path, err)
	}

	fsType := string(mount.FSType)

	err = syscall.Mount(fsType, mount.Path, fsType, 0, "")
	if err != nil {
		return fmt.Errorf("mount %s (%s): %v", mount.Path, mount.FSType, err)
	}

	return nil
}

// MountAll mounts all known essential special file systems at the usual paths.
//
// All special file systems required for usual operations, like accessing
// kernel variables, modifying kernel knobs or accessing devices are mounted.
func MountAll(mountPoints MountPoints) error {
	for _, mp := range mountPoints {
		if err := Mount(mp); err != nil {
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
			return fmt.Errorf("create common symlink %s: %v", link, err)
		}
	}

	return nil
}
