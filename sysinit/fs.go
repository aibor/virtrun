// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package sysinit

import (
	"fmt"
	"os"
	"runtime"
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

// MountPointPresets defines architecture specific mount points.
var MountPointPresets = map[string]MountPoints{
	"amd64": {
		{"/proc", FSTypeProc},
		{"/sys", FSTypeSys},
		{"/sys/fs/bpf", FSTypeBpf},
		{"/sys/kernel/tracing", FSTypeTracing},
		{"/dev", FSTypeDevTmp},
		{"/run", FSTypeTmp},
		{"/tmp", FSTypeTmp},
	},
	"arm64": {
		{"/proc", FSTypeProc},
		{"/sys", FSTypeSys},
		{"/sys/fs/bpf", FSTypeBpf},
		{"/dev", FSTypeDevTmp},
		{"/run", FSTypeTmp},
		{"/tmp", FSTypeTmp},
	},
}

// CommonSymlinks defines symbolic links usually set by init systems.
var CommonSymlinks = map[string]string{
	"/dev/fd":     "/proc/self/fd/",
	"/dev/stdin":  "/proc/self/fd/0",
	"/dev/stdout": "/proc/self/fd/1",
	"/dev/stderr": "/proc/self/fd/2",
}

// MountFs mounts the special file system with type FsType at the given path.
//
// If path does not exist, it is created. An error is returned if this or the
// mount syscall fails.
func MountFs(path string, fstype FSType) error {
	if err := os.MkdirAll(path, defaultDirMode); err != nil {
		return fmt.Errorf("mkdir %s: %v", path, err)
	}

	if err := syscall.Mount(string(fstype), path, string(fstype), 0, ""); err != nil {
		return fmt.Errorf("mount %s (%s): %v", path, fstype, err)
	}

	return nil
}

// MountAll mounts all known essential special file systems at the usual paths.
//
// All special file systems required for usual operations, like accessing
// kernel variables, modifying kernel knobs or accessing devices are mounted.
func MountAll() error {
	arch := runtime.GOARCH

	mounts, exists := MountPointPresets[arch]
	if !exists {
		return fmt.Errorf("no mount point preset found for arch %s", arch)
	}

	for _, mp := range mounts {
		if err := MountFs(mp.Path, mp.FSType); err != nil {
			return err
		}
	}

	return nil
}

// CreateCommonSymlinks creates common symbolic links in the file system.
//
// This must be run after all file systems have been mounted.
func CreateCommonSymlinks() error {
	for link, target := range CommonSymlinks {
		if err := os.Symlink(target, link); err != nil {
			return fmt.Errorf("create common symlink %s: %v", link, err)
		}
	}

	return nil
}
