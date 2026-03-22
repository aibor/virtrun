// SPDX-FileCopyrightText: 2026 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build linux

package sysinit

import (
	"errors"
	"fmt"

	"golang.org/x/sys/unix"
)

type ifreqAddrKind uint

const (
	ifreqAddrLocal ifreqAddrKind = iota
	ifreqAddrNetmask
)

type ifaceControlHandle struct {
	socketFD int
	ifreq    *unix.Ifreq
}

func newIfaceRequestHandle(ifname string) (*ifaceControlHandle, error) {
	unixIfreq, err := unix.NewIfreq(ifname)
	if err != nil {
		if errors.Is(err, unix.EINVAL) {
			err = ErrInvalidIfaceName
		}

		return nil, fmt.Errorf("new ifreq: %w", err)
	}

	sock, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, 0)
	if err != nil {
		return nil, fmt.Errorf("socket: %w", err)
	}

	ifreqSock := ifaceControlHandle{
		socketFD: sock,
		ifreq:    unixIfreq,
	}

	return &ifreqSock, nil
}

func (h ifaceControlHandle) Close() error {
	return fclose(h.socketFD)
}

func (h ifaceControlHandle) ioctl(req uint) error {
	err := unix.IoctlIfreq(h.socketFD, req, h.ifreq)
	if err != nil {
		return fmt.Errorf("ioctl: %w", err)
	}

	return nil
}

func (h ifaceControlHandle) setAddr(kind ifreqAddrKind, addr []byte) error {
	err := h.ifreq.SetInet4Addr(addr)
	if err != nil {
		return fmt.Errorf("marshal addr: %w", err)
	}

	var req uint

	switch kind {
	case ifreqAddrNetmask:
		req = unix.SIOCSIFNETMASK
	default:
		req = unix.SIOCSIFADDR
	}

	return h.ioctl(req)
}

func (h ifaceControlHandle) updateFlags(flagsFn ifaceFlagsFunc) error {
	err := h.ioctl(unix.SIOCGIFFLAGS)
	if err != nil {
		return fmt.Errorf("get flags: %w", err)
	}

	flags := h.ifreq.Uint16()

	flagsFn(&flags)

	h.ifreq.SetUint16(flags)

	err = h.ioctl(unix.SIOCSIFFLAGS)
	if err != nil {
		return fmt.Errorf("set flags: %w", err)
	}

	return nil
}

type ifaceFlagsFunc func(flags *uint16)

func ifaceRequestFlagsSetUp(flags *uint16) {
	*flags |= unix.IFF_UP | unix.IFF_RUNNING
}
