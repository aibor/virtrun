// SPDX-FileCopyrightText: 2026 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd_test

import (
	"strconv"
	"testing"

	"github.com/aibor/virtrun/internal/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLimitedUintValue_Set(t *testing.T) {
	ptr := func(n uint64) *uint64 {
		return &n
	}

	tests := []struct {
		name        string
		value       cmd.LimitedUintValue
		input       string
		expected    *uint64
		expectedErr error
	}{
		{
			name:        "empty",
			expectedErr: strconv.ErrSyntax,
		},
		{
			name:        "not a number",
			input:       "dwdfwef",
			expectedErr: strconv.ErrSyntax,
		},
		{
			name:        "signed int",
			input:       "-1",
			expectedErr: strconv.ErrSyntax,
		},
		{
			name:        "longer than 64bit",
			input:       "184467440737095516151111111111111111111",
			expectedErr: strconv.ErrRange,
		},
		{
			name:  "zero",
			input: "0",
			value: cmd.LimitedUintValue{
				Value: ptr(42),
			},
			expected: ptr(0),
		},
		{
			name:  "is lower",
			input: "42",
			value: cmd.LimitedUintValue{
				Value: ptr(0),
				Lower: 42,
				Upper: 43,
			},
			expected: ptr(42),
		},
		{
			name:  "is upper",
			input: "42",
			value: cmd.LimitedUintValue{
				Value: ptr(0),
				Lower: 41,
				Upper: 42,
			},
			expected: ptr(42),
		},
		{
			name:  "is below",
			input: "42",
			value: cmd.LimitedUintValue{
				Lower: 43,
				Upper: 44,
			},
			expectedErr: cmd.ErrValueOutOfRange,
		},
		{
			name:  "is above",
			input: "42",
			value: cmd.LimitedUintValue{
				Lower: 40,
				Upper: 41,
			},
			expectedErr: cmd.ErrValueOutOfRange,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.value.Set(tt.input)
			require.ErrorIs(t, err, tt.expectedErr)
			assert.Equal(t, tt.expected, tt.value.Value)
		})
	}
}
