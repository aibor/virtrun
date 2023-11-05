package pidonetest

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

// MountFs mounts the special file system with type FsType at the given path.
//
// If path does not exist, it is created. An error is returned if this or the
// mount syscall fails.
func MountFs(path string, fstype FSType) error {
	if err := os.MkdirAll(path, 0755); err != nil {
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
