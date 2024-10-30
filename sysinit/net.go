// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// ConfigureLoopbackInterface brings the loopback interface up.
//
// Kernel should configure address already automatically.
func ConfigureLoopbackInterface() error {
	// Any socket can be used for sending ioctls.
	sock, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, 0)
	if err != nil {
		return fmt.Errorf("create control socket: %w", err)
	}

	ifReq, err := unix.NewIfreq("lo")
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}

	ifReq.SetUint16(unix.IFF_UP)

	err = unix.IoctlIfreq(sock, unix.SIOCSIFFLAGS, ifReq)
	if err != nil {
		return fmt.Errorf("ioctl: %w", err)
	}

	return nil
}
