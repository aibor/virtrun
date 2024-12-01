// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package qemu

import (
	"fmt"
	"slices"
)

const (
	// TransportTypeISA is ISA legacy transport. It should work for amd64 in
	// any case. With "microvm" machine type only provides one console for
	// stdout.
	TransportTypeISA TransportType = "isa"
	// TransportTypePCI is VirtIO PCI transport. Requires kernel built with
	// CONFIG_VIRTIO_PCI.
	TransportTypePCI TransportType = "pci"
	// TransportTypeMMIO is Virtio MMIO transport. Requires kernel built with
	// CONFIG_VIRTIO_MMIO.
	TransportTypeMMIO TransportType = "mmio"
)

// TransportType represents QEMU IO transport types.
type TransportType string

func (t *TransportType) isKnown() bool {
	knownTransportTypes := []TransportType{
		TransportTypeISA,
		TransportTypePCI,
		TransportTypeMMIO,
	}

	return slices.Contains(knownTransportTypes, *t)
}

// String returns the [TransportType]'s underlying string value.
//
// It returns the empty string for unknown [TransportType]s.
func (t *TransportType) String() string {
	if !t.isKnown() {
		return ""
	}

	return string(*t)
}

// Set parses the given string and sets the receiving [TransportType].
//
// It returns ErrTransportTypeInvalid if the string does not represent a valid
// [TransportType].
func (t *TransportType) Set(s string) error {
	tt := TransportType(s)

	if !tt.isKnown() {
		return ErrTransportTypeInvalid
	}

	*t = tt

	return nil
}

// ConsoleDeviceName returns the name of the console device in the guest.
func (t *TransportType) ConsoleDeviceName(num uint) string {
	f := "hvc%d"
	if *t == TransportTypeISA {
		f = "ttyS%d"
	}

	return fmt.Sprintf(f, num)
}

func prepareConsoleArgs(transportType TransportType) []Argument {
	switch transportType {
	case TransportTypePCI:
		return []Argument{
			RepeatableArg("device", "virtio-serial-pci,max_ports=8"),
		}
	case TransportTypeMMIO:
		return []Argument{
			RepeatableArg("device", "virtio-serial-device,max_ports=8"),
		}
	default: // Ignore invalid transport types.
		return nil
	}
}

func consoleArgsFunc(transportType TransportType) func(int) []Argument {
	switch transportType {
	case TransportTypeISA:
		return func(fd int) []Argument {
			return []Argument{
				RepeatableArg("serial", "file:"+fdPath(fd)),
			}
		}
	case TransportTypePCI, TransportTypeMMIO:
		return func(fd int) []Argument {
			vcon := fmt.Sprintf("vcon%d", fd)
			chardev := fmt.Sprintf("file,id=%s,path=%s", vcon, fdPath(fd))
			device := "virtconsole,chardev=" + vcon

			return []Argument{
				RepeatableArg("chardev", chardev),
				RepeatableArg("device", device),
			}
		}
	default: // Ignore invalid transport types.
		return func(_ int) []Argument { return nil }
	}
}

func fdPath(fd int) string {
	return fmt.Sprintf("/dev/fd/%d", fd)
}
