// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package exitcode_test

import (
	"fmt"
	"testing"

	"github.com/aibor/virtrun/internal/exitcode"
	"github.com/stretchr/testify/assert"
)

func TestError_Is(t *testing.T) {
	tests := []struct {
		name   string
		other  error
		assert assert.BoolAssertionFunc
	}{
		{
			name:   "nil",
			assert: assert.False,
		},
		{
			name:   "same",
			other:  exitcode.Error(42),
			assert: assert.True,
		},
		{
			name:   "other",
			other:  assert.AnError,
			assert: assert.False,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := exitcode.Error(0)
			tt.assert(t, err.Is(tt.other))
		})
	}
}

func TestError_Code(t *testing.T) {
	err := exitcode.Error(42)
	assert.Equal(t, 42, err.Code())
}

func TestFrom(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		expected    int
		assertIsErr assert.BoolAssertionFunc
	}{
		{
			name:        "no error",
			expected:    0,
			assertIsErr: assert.False,
		},
		{
			name:        "an error",
			err:         assert.AnError,
			expected:    -1,
			assertIsErr: assert.False,
		},
		{
			name:        "exit error",
			err:         exitcode.Error(42),
			expected:    42,
			assertIsErr: assert.True,
		},
		{
			name:        "wrapped exit error",
			err:         fmt.Errorf("test: %w", exitcode.Error(42)),
			expected:    42,
			assertIsErr: assert.True,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, isExitErr := exitcode.From(tt.err)

			assert.Equal(t, tt.expected, actual)
			tt.assertIsErr(t, isExitErr)
		})
	}
}
