// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"syscall"
)

const (
	activeConsoleFile = "/sys/devices/virtual/tty/console/active"
	infoFileDir       = "/proc/tty/driver/"
)

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
	primaryConsole, err := primaryConsole()
	if err != nil {
		return nil, err
	}

	switch {
	case strings.HasPrefix(primaryConsole, "hvc"):
		return virtConsolesConnected()
	case strings.HasPrefix(primaryConsole, "ttyS"):
		return connectedTTYConsoles("ttyS", "serial")
	case strings.HasPrefix(primaryConsole, "ttyAMA"):
		return connectedTTYConsoles("ttyAMA", "ttyAMA")
	}

	return nil, fmt.Errorf("%w: %s", ErrConsoleNotSupported, primaryConsole)
}

// virtConsolesConnected returns a slice of virtio consoles (/dev/hvc*) that are
// connected on the host.
func virtConsolesConnected() ([]console, error) {
	consoles := []console{}

	files, err := fs.Glob(os.DirFS("/dev/"), "hvc*")
	if err != nil {
		return nil, fmt.Errorf("read hvc entries: %w", err)
	}

	for _, file := range files {
		path := "/dev/" + file

		hvc, err := os.Open(path)
		if err != nil {
			// virtio consoles that are not connected on the host return ENODEV.
			if errors.Is(err, syscall.ENODEV) {
				continue
			}

			return nil, fmt.Errorf("check: %w", err)
		}

		_ = hvc.Close()

		port, err := strconv.Atoi(strings.TrimPrefix(file, "hvc"))
		if err != nil {
			return nil, fmt.Errorf("parse port: %w", err)
		}

		consoles = append(consoles, console{
			path: path,
			port: port,
		})
	}

	return consoles, nil
}

// connectedTTYConsoles returns a slice of consoles that are connected on the
// host.
func connectedTTYConsoles(typ string, driver string) ([]console, error) {
	ports, err := connectedTTYs(driver)
	if err != nil {
		return nil, err
	}

	consoles := []console{}
	for _, port := range ports {
		consoles = append(consoles, console{
			path: consolePath(typ, port),
			port: port,
		})
	}

	return consoles, nil
}

// connectedTTYs returns a slice of tty port numbers that are connected on the
// host with the given driver.
func connectedTTYs(driver string) ([]int, error) {
	serialInfo, err := os.ReadFile(infoFileDir + driver)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []int{}, nil
		}

		return nil, fmt.Errorf("read info: %w", err)
	}

	consoles := connectedTTYsFromInfo(serialInfo)

	return consoles, nil
}

func connectedTTYsFromInfo(serialInfo []byte) []int {
	ports := []int{}

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

		ports = append(ports, port)
	}

	return ports
}

func consolePath(typ string, id int) string {
	return "/dev/" + typ + strconv.Itoa(id)
}

func primaryConsole() (string, error) {
	active, err := os.ReadFile(activeConsoleFile)
	if err != nil {
		return "", fmt.Errorf("read active console: %w", err)
	}

	return string(bytes.Fields(active)[0]), nil
}
