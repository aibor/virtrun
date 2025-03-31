// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package qemu_test

import (
	"testing"

	"github.com/aibor/virtrun/internal/qemu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommmandConsoleDeviceName(t *testing.T) {
	tests := []struct {
		id        uint
		transport qemu.TransportType
		expect    string
	}{
		{
			id:        5,
			transport: qemu.TransportTypeISA,
			expect:    "ttyS5",
		},
		{
			id:        3,
			transport: qemu.TransportTypePCI,
			expect:    "hvc3",
		},
		{
			id:        1,
			transport: qemu.TransportTypeMMIO,
			expect:    "hvc1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.expect, func(t *testing.T) {
			assert.Equal(t, tt.expect, tt.transport.ConsoleDeviceName(tt.id))
		})
	}
}

func TestTransportType_String(t *testing.T) {
	tests := []struct {
		input    qemu.TransportType
		expected string
	}{
		{
			input:    qemu.TransportTypeISA,
			expected: "isa",
		},
		{
			input:    qemu.TransportTypePCI,
			expected: "pci",
		},
		{
			input:    qemu.TransportTypeMMIO,
			expected: "mmio",
		},
		{
			input:    qemu.TransportType("unknown"),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			actual := tt.input.String()
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestTransportType_Set(t *testing.T) {
	tests := []struct {
		input       string
		expected    qemu.TransportType
		expectedErr error
	}{
		{
			input:    "isa",
			expected: qemu.TransportTypeISA,
		},
		{
			input:    "pci",
			expected: qemu.TransportTypePCI,
		},
		{
			input:    "mmio",
			expected: qemu.TransportTypeMMIO,
		},
		{
			input:       "unknown",
			expectedErr: qemu.ErrTransportTypeInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var actual qemu.TransportType

			err := actual.Set(tt.input)
			require.ErrorIs(t, err, tt.expectedErr)

			assert.Equal(t, tt.expected, actual)
		})
	}
}
