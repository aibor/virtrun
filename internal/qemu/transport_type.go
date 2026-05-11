// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package qemu

import (
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
	ttype := TransportType(s)

	if !ttype.isKnown() {
		return ErrTransportTypeInvalid
	}

	*t = ttype

	return nil
}

func (t *TransportType) isKnown() bool {
	knownTransportTypes := []TransportType{
		TransportTypeISA,
		TransportTypePCI,
		TransportTypeMMIO,
	}

	return slices.Contains(knownTransportTypes, *t)
}
