// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSerialConsolesConnectedFromBytes(t *testing.T) {
	tests := []struct {
		name        string
		input       []string
		expected    []console
		expectedErr error
	}{
		{
			name:     "empty",
			expected: []console{},
		},
		{
			name: "single console",
			input: []string{
				"serinfo:1.0 driver revision:\n",
				"0: uart:16550A port:000003F8 irq:4 tx:0 rx:0 RTS|CTS|DSR|CD\n",
				"1: uart:unknown port:000002F8 irq:3\n",
				"2: uart:unknown port:000003E8 irq:4\n",
				"3: uart:unknown port:000002E8 irq:3\n",
				"4: uart:unknown port:00000000 irq:0\n",
			},
			expected: []console{
				{port: 0, path: "/dev/ttyS0"},
			},
		},
		{
			name: "multiple consoles",
			input: []string{
				"serinfo:1.0 driver revision:\n",
				"0: uart:16550A port:000003F8 irq:4 tx:0 rx:0 RTS|CTS|DSR|CD\n",
				"1: uart:16550A port:000002F8 irq:3 tx:0 rx:0 CTS|DSR|CD\n",
				"2: uart:16550A port:000003E8 irq:4 tx:0 rx:0 CTS|DSR|CD\n",
				"3: uart:unknown port:000002E8 irq:3\n",
				"4: uart:unknown port:00000000 irq:0\n",
			},
			expected: []console{
				{port: 0, path: "/dev/ttyS0"},
				{port: 1, path: "/dev/ttyS1"},
				{port: 2, path: "/dev/ttyS2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := []byte(strings.Join(tt.input, ""))
			actual, err := serialConsolesConnectedFromBytes(input)
			require.ErrorIs(t, err, tt.expectedErr)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestConsolePath(t *testing.T) {
	tests := []struct {
		name     string
		typ      string
		port     int
		expected string
	}{
		{
			name:     "virtio",
			typ:      "hvc",
			port:     42,
			expected: "/dev/hvc42",
		},
		{
			name:     "serial",
			typ:      "ttyS",
			port:     269,
			expected: "/dev/ttyS269",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := consolePath(tt.typ, tt.port)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
