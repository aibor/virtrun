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

func TestTransportType_ConsoleDeviceName(t *testing.T) {
	tests := []struct {
		input    qemu.TransportType
		expected string
	}{
		{
			input:    qemu.TransportTypeISA,
			expected: "ttyS1",
		},
		{
			input:    qemu.TransportTypePCI,
			expected: "hvc1",
		},
		{
			input:    qemu.TransportTypeMMIO,
			expected: "hvc1",
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			actual := tt.input.ConsoleDeviceName(1)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
