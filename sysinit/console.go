// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strconv"
)

const serialConsoleInfoFile = "/proc/tty/driver/serial"

type console struct {
	path string
	port int
}

// connectedConsoles returns a slice of consoles that are detected as
// connected on the host.
//
// If virto consoles are present (/dev/hvc*) then only those are used. Otherwise
// serial consoles (/dev/ttyS*) are used.
func connectedConsoles() ([]console, error) {
	// If virtio consoles are present, use these.
	consoles := virtConsolesConnected()
	if len(consoles) > 0 {
		return consoles, nil
	}

	// Otherwise fall back to serial consoles.
	consoles, err := serialConsolesConnected()
	if err != nil {
		return nil, fmt.Errorf("serial: %w", err)
	}

	return consoles, nil
}

// virtConsolesConnected returns a slice of virtio consoles (/dev/hvc*) that are
// connected on the host.
func virtConsolesConnected() []console {
	consoles := []console{}

	//nolint:lll
	// https://github.com/torvalds/linux/blob/dd9c17322a6cc56d57b5d2b0b84393ab76a55c80/drivers/tty/hvc/hvc_console.h#L33
	for port := range 8 {
		path := consolePath("hvc", port)

		hvc, err := os.Open(path)
		if err != nil {
			// If the file is not present and there are no consoles yet,
			// there are no virtio consoles present at all.
			if errors.Is(err, os.ErrNotExist) && len(consoles) == 0 {
				return nil
			}

			// virtio consoles that are not connected on the host return ENODEV.
			continue
		}

		_ = hvc.Close()

		consoles = append(consoles, console{
			path: path,
			port: port,
		})
	}

	return consoles
}

// serialConsolesConnected returns a slice of serial consoles (/dev/ttyS*) that
// are connected on the host using the default serial driver info file.
func serialConsolesConnected() ([]console, error) {
	serialInfo, err := os.ReadFile(serialConsoleInfoFile)
	if err != nil {
		return nil, fmt.Errorf("read info: %w", err)
	}

	return serialConsolesConnectedFromBytes(serialInfo)
}

// serialConsolesConnectedFromBytes returns a slice of serial consoles
// (/dev/ttyS*) that are connected on the host from the given reader.
//
// The reader is expected to have the default serial info file format.
func serialConsolesConnectedFromBytes(serialInfo []byte) ([]console, error) {
	consoles := []console{}

	for line := range bytes.Lines(serialInfo) {
		if !bytes.Contains(line, []byte("uart")) ||
			bytes.Contains(line, []byte("uart:unknown")) {
			continue
		}

		// The serial driver info lists port information. File layout:
		//	serinfo:1.0 driver revision:
		// 	0: uart:16550A port:000003F8 irq:4 tx:126 rx:0 RTS|CTS|DTR|DSR|CD
		// 	1: uart:unknown port:000002F8 irq:3
		// 	...
		portField, _, found := bytes.Cut(line, []byte(":"))
		if !found {
			continue
		}

		port, err := strconv.Atoi(string(portField))
		if err != nil {
			continue
		}

		consoles = append(consoles, console{
			path: consolePath("ttyS", port),
			port: port,
		})
	}

	return consoles, nil
}

func consolePath(typ string, id int) string {
	return "/dev/" + typ + strconv.Itoa(id)
}
