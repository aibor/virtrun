// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package exitcode_test

import (
	"bytes"
	"testing"

	"github.com/aibor/virtrun/internal/exitcode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFprint(t *testing.T) {
	var actual bytes.Buffer

	actualWritten, err := exitcode.Fprint(&actual, 42)
	require.NoError(t, err)

	assert.Equal(t, exitcode.Identifier+": 42\n", actual.String())
	assert.Equal(t, len(exitcode.Identifier)+5, actualWritten)
}

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
			actual, found := exitcode.Parse(tt.input)
			tt.assertFound(t, found)

			assert.Equal(t, tt.expected, actual)
		})
	}
}
