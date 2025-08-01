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
	FSTypeTracing  FSType = "tracefs"

	defaultDirMode = 0o755
)

// EssentialMountPoints returns a map of all essential special pseudo and
// virtual file systems required for usual system operations, like accessing
// kernel variables, modifying kernel knobs or accessing devices.
func essentialMountPoints() MountPoints {
	return MountPoints{
		"/dev": {
			FSType: FSTypeDevTmp,
			Flags:  MS_NOSUID,
		},
		"/proc": {
			FSType: FSTypeProc,
			Flags:  MS_NOSUID | MS_NODEV | MS_NOEXEC,
		},
		"/sys": {
			FSType: FSTypeSys,
			Flags:  MS_NOSUID | MS_NODEV | MS_NOEXEC,
		},
	}
}

// SystemMountPoints returns a map of non-essential special pseudo and virtual
// file systems.
func SystemMountPoints() MountPoints {
	return MountPoints{
		"/dev/hugepages":           {FSType: FSTypeHugeTlb},
		"/dev/mqueue":              {FSType: FSTypeMqueue},
		"/dev/pts":                 {FSType: FSTypeDevPts},
		"/sys/fs/bpf":              {FSType: FSTypeBpf},
		"/sys/fs/cgroup":           {FSType: FSTypeCgroup2},
		"/sys/fs/fuse/connections": {FSType: FSTypeFuseCtl},
		"/sys/fs/pstore":           {FSType: FSTypePstore},
		"/sys/kernel/config":       {FSType: FSTypeConfig},
		"/sys/kernel/debug":        {FSType: FSTypeDebug},
		"/sys/kernel/security":     {FSType: FSTypeSecurity},
		"/sys/kernel/tracing":      {FSType: FSTypeTracing},
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
	return func(_ *State) error {
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
