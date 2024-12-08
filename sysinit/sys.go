// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"errors"
	"fmt"

	"golang.org/x/sys/unix"
)

type finitFlags int

const finitFlagCompressedFile finitFlags = unix.MODULE_INIT_COMPRESSED_FILE

func mount(path, source, fsType string) error {
	if source == "" {
		source = fsType
	}

	if err := unix.Mount(source, path, fsType, 0, ""); err != nil {
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
