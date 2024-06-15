// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package qemu

import (
	"fmt"
)

const (
	// ISA legacy transport. Should work for amd64 in any case. With "microvm"
	// machine type only provides one console for stdout.
	TransportTypeISA TransportType = iota
	// Virtio PCI transport. Requires kernel built with CONFIG_VIRTIO_PCI.
	TransportTypePCI
	// Virtio MMIO transport. Requires kernel built with CONFIG_VIRTIO_MMIO.
	TransportTypeMMIO

	LenTransportType
)

// TransportType represents QEMU IO transport types.
type TransportType int

// ConsoleDeviceName returns the name of the console device in the guest.
func (t *TransportType) ConsoleDeviceName(num uint8) string {
	f := "hvc%d"
	if *t == TransportTypeISA {
		f = "ttyS%d"
	}

	return fmt.Sprintf(f, num)
}
