// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package exitcode_test

import (
	"testing"

	"github.com/aibor/virtrun/internal/exitcode"
	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
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
			input:       exitcode.Identifier + ": 0",
			assertFound: assert.True,
		},
		{
			name:        "matching input",
			input:       exitcode.Identifier + ": 42",
			expected:    42,
			assertFound: assert.True,
		},
		{
			name:        "matching input with trailing",
			input:       exitcode.Identifier + ": 42 whatever",
			expected:    42,
			assertFound: assert.True,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, found := exitcode.Parse([]byte(tt.input))
			tt.assertFound(t, found)

			assert.Equal(t, tt.expected, actual)
		})
	}
}
