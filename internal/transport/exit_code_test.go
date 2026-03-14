// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package transport_test

import (
	"testing"

	"github.com/aibor/virtrun/internal/transport"
	"github.com/stretchr/testify/assert"
)

func TestFormatExitCode(t *testing.T) {
	tests := []struct {
		exitcode int
		expected string
	}{
		{0, transport.Identifier + ": 0"},
		{1, transport.Identifier + ": 1"},
		{42, transport.Identifier + ": 42"},
		{-42, transport.Identifier + ": -42"},
	}

	for _, tt := range tests {
		actual := transport.FormatExitCode(tt.exitcode)
		assert.Equal(t, tt.expected, actual)
	}
}

func TestParseExitCode(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    int
		assertFound assert.BoolAssertionFunc
	}{
		{
			name:        "empty input",
			assertFound: assert.False,
		},
		{
			name:        "matching input zero",
			input:       transport.Identifier + ": 0",
			assertFound: assert.True,
		},
		{
			name:        "matching input",
			input:       transport.Identifier + ": 42",
			expected:    42,
			assertFound: assert.True,
		},
		{
			name:        "matching input with trailing",
			input:       transport.Identifier + ": 42 whatever",
			expected:    42,
			assertFound: assert.True,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, found := transport.ParseExitCode([]byte(tt.input))
			tt.assertFound(t, found)

			assert.Equal(t, tt.expected, actual)
		})
	}
}
