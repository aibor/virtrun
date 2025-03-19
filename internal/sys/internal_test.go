// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sys

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func AssertContainsPaths(tb testing.TB, expected, actual []string) bool {
	tb.Helper()

	makeAbs := func(in []string) []string {
		var out []string

		for _, path := range in {
			abs, err := filepath.Abs(path)
			require.NoError(tb, err)

			out = append(out, abs)
		}

		return out
	}

	actualAbs := makeAbs(actual)
	expectedAbs := makeAbs(expected)

	assert.Subset(tb, actualAbs, expectedAbs)

	return true
}
