// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build linux

package sysinit

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

//revive:disable:var-naming

// Mount flags.
const (
	MS_NOSUID = unix.MS_NOSUID
	MS_NODEV  = unix.MS_NODEV
	MS_NOEXEC = unix.MS_NOEXEC
)

//revive:enable:var-naming

// MountFlags is a set of flags passed to the [unix.Mount] syscall.
type MountFlags int

func mount(path, source, fsType string, flags MountFlags, data string) error {
	if source == "" {
		source = fsType
	}

	err := unix.Mount(source, path, fsType, uintptr(flags), data)
	if err != nil {
		return fmt.Errorf("mount %s: %w", path, err)
	}

	return nil
}

func initModule(data []byte, params string) error {
	if err := unix.InitModule(data, params); err != nil {
		return fmt.Errorf("init_module: %w", err)
	}

	return nil
}

type finitFlags int

const finitFlagCompressedFile finitFlags = unix.MODULE_INIT_COMPRESSED_FILE

func finitModule(fd int, params string, flags finitFlags) error {
	if err := unix.FinitModule(fd, params, int(flags)); err != nil {
		// If finit_module is not available, EOPNOTSUPP is returned.
		if errors.Is(err, unix.EOPNOTSUPP) {
			err = errors.ErrUnsupported
		}

		return fmt.Errorf("finit_module: %w", err)
	}

	return nil
}

func reboot() error {
	if err := unix.Reboot(unix.LINUX_REBOOT_CMD_RESTART); err != nil {
		return fmt.Errorf("reboot: %w", err)
	}

	return nil
}

func setInterfaceUp(name string) error {
	sock, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, 0)
	if err != nil {
		return fmt.Errorf("control socket: %w", err)
	}

	ifReq, err := unix.NewIfreq(name)
	if err != nil {
		return fmt.Errorf("interface request: %w", err)
	}

	ifReq.SetUint16(unix.IFF_UP)

	if err := unix.IoctlIfreq(sock, unix.SIOCSIFFLAGS, ifReq); err != nil {
		return fmt.Errorf("ioctl: %w", err)
	}

	return nil
}

func sysctl(key, value string) error {
	const mode = 0o600

	path := "/proc/sys/" + key

	if err := os.WriteFile(path, []byte(value), mode); err != nil {
		return fmt.Errorf("sysctl %s: %w", key, err)
	}

	return nil
}

func getpid() int {
	return unix.Getpid()
}

func setenv(key, value string) error {
	if err := unix.Setenv(key, value); err != nil {
		return fmt.Errorf("setenv %s: %w", key, err)
	}

	return nil
}
