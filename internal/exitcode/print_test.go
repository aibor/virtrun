// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package exitcode_test

import (
	"testing"

	"github.com/aibor/virtrun/internal/exitcode"
	"github.com/stretchr/testify/assert"
)

func TestSprint(t *testing.T) {
	tests := []struct {
		exitcode int
		expected string
	}{
		{0, exitcode.Identifier + ": 0"},
		{1, exitcode.Identifier + ": 1"},
		{42, exitcode.Identifier + ": 42"},
		{-42, exitcode.Identifier + ": -42"},
	}

	for _, tt := range tests {
		actual := exitcode.Sprint(tt.exitcode)
		assert.Equal(t, tt.expected, actual)
	}
}
