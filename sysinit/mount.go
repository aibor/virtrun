package sysinit

import (
	"fmt"
	"os"
	"syscall"
)

// FsType is a file system type.
type FsType string

// Special essential file system types.
const (
	FsTypeDevTmp  FsType = "devtmpfs"
	FsTypeProc    FsType = "proc"
	FsTypeSys     FsType = "sysfs"
	FsTypeTmp     FsType = "tmpfs"
	FsTypeBpf     FsType = "bpf"
	FsTypeTracing FsType = "tracefs"
)

// MountFs mounts the special file system with type FsType at the given path.
//
// If path does not exist, it is created. An error is returned if this or the
// mount syscall fails.
func MountFs(path string, fstype FsType) error {
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
	mounts := []struct {
		path   string
		fstype FsType
	}{
		{"/proc", FsTypeProc},
		{"/sys", FsTypeSys},
		{"/sys/fs/bpf", FsTypeBpf},
		{"/sys/kernel/tracing", FsTypeTracing},
		{"/dev", FsTypeDevTmp},
		{"/run", FsTypeTmp},
		{"/tmp", FsTypeTmp},
	}

	for _, mp := range mounts {
		if err := MountFs(mp.path, mp.fstype); err != nil {
			return err
		}
	}

	return nil
}
