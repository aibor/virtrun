// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package qemu

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsoleProcessor_Run(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectedErr error
	}{
		{
			name: "empty",
		},
		{
			name:     "crlf only",
			input:    "\r\n",
			expected: "\n",
		},
		{
			name:     "lf only",
			input:    "\n",
			expected: "\n",
		},
		{
			name:     "with crlf",
			input:    "some first\r\nand second\r\nand third line",
			expected: "some first\nand second\nand third line\n",
		},
		{
			name:     "with lf",
			input:    "some first\nand second\nand third line",
			expected: "some first\nand second\nand third line\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer

			processor := consoleProcessor{
				dst: &output,
				src: bytes.NewBufferString(tt.input),
			}

			err := processor.run()
			require.ErrorIs(t, err, tt.expectedErr)

			assert.Equal(t, tt.expected, output.String())
		})
	}
}
