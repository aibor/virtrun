// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"errors"
	"fmt"
	"log"
	"os"
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

// SystemMountPoints returns a map of all special pseudo and virtual file
// systems required for usual system operations, like accessing kernel
// variables, modifying kernel knobs or accessing devices.
func SystemMountPoints() MountPoints {
	return MountPoints{
		"/dev":                     {FSType: FSTypeDevTmp},
		"/dev/hugepages":           {FSType: FSTypeHugeTlb, MayFail: true},
		"/dev/mqueue":              {FSType: FSTypeMqueue, MayFail: true},
		"/dev/pts":                 {FSType: FSTypeDevPts, MayFail: true},
		"/dev/shm":                 {FSType: FSTypeTmp, MayFail: true},
		"/proc":                    {FSType: FSTypeProc},
		"/run":                     {FSType: FSTypeTmp},
		"/sys/fs/bpf":              {FSType: FSTypeBpf, MayFail: true},
		"/sys/fs/cgroup":           {FSType: FSTypeCgroup2, MayFail: true},
		"/sys/fs/fuse/connections": {FSType: FSTypeFuseCtl, MayFail: true},
		"/sys/fs/pstore":           {FSType: FSTypePstore, MayFail: true},
		"/sys":                     {FSType: FSTypeSys},
		"/sys/kernel/config":       {FSType: FSTypeConfig, MayFail: true},
		"/sys/kernel/debug":        {FSType: FSTypeDebug, MayFail: true},
		"/sys/kernel/security":     {FSType: FSTypeSecurity, MayFail: true},
		"/sys/kernel/tracing":      {FSType: FSTypeTracing, MayFail: true},
		"/tmp":                     {FSType: FSTypeTmp},
	}
}

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

// MountPoints is a collection of MountPoints.
type MountPoints map[string]MountOptions

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

// MountAll mounts the given set of system file systems.
//
// The mounts are executed in lexicographic order of the paths. If only
// optional mount points failed, it returns an [OptionalMountError] with all
// errors.
func MountAll(mountPoints MountPoints) error {
	var optionalErrs OptionalMountError

	for path, opts := range sortedMap(mountPoints) {
		if err := Mount(path, opts); err != nil {
			if !opts.MayFail {
				return err
			}

			optionalErrs = append(optionalErrs, err)
		}
	}

	if optionalErrs != nil {
		return optionalErrs
	}

	return nil
}

// WithMountPoints returns a setup [Func] that wraps [MountAll] and can be used
// with [Run].
//
// It logs optional mounts that failed.
func WithMountPoints(mountPoints MountPoints) Func {
	return func() error {
		err := MountAll(mountPoints)

		var optionalErrs OptionalMountError
		if errors.As(err, &optionalErrs) {
			for _, err := range optionalErrs {
				log.Println("INFO optional mount failed: ", err.Error())
			}

			return nil
		}

		return err
	}
}
