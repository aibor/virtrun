// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package qemu_test

import (
	"testing"

	"github.com/aibor/virtrun/internal/qemu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommmandConsoleDeviceName(t *testing.T) {
	tests := []struct {
		id            uint8
		transportType qemu.TransportType
		expect        string
	}{
		{
			id:            5,
			transportType: qemu.TransportTypeISA,
			expect:        "ttyS5",
		},
		{
			id:            3,
			transportType: qemu.TransportTypePCI,
			expect:        "hvc3",
		},
		{
			id:            1,
			transportType: qemu.TransportTypeMMIO,
			expect:        "hvc1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.expect, func(t *testing.T) {
			assert.Equal(t, tt.expect, tt.transportType.ConsoleDeviceName(tt.id))
		})
	}
}

func TestTransportType_MarshalText(t *testing.T) {
	tests := []struct {
		input       qemu.TransportType
		expected    string
		expectedErr error
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
			input:       qemu.TransportType("unknown"),
			expectedErr: qemu.ErrTransportTypeInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			actual, err := tt.input.MarshalText()
			require.ErrorIs(t, err, tt.expectedErr)

			assert.Equal(t, tt.expected, string(actual))
		})
	}
}

func TestTransportType_UnmarshalText(t *testing.T) {
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

			err := actual.UnmarshalText([]byte(tt.input))
			require.ErrorIs(t, err, tt.expectedErr)

			assert.Equal(t, tt.expected, actual)
		})
	}
}
