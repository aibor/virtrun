// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sys

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func AssertContainsPaths(tb testing.TB, actual, expected []string) bool {
	tb.Helper()

	expectedAbs := make(map[string]string, len(expected))

	for _, path := range expected {
		abs, err := filepath.Abs(path)
		require.NoErrorf(tb, err, "must absolute path %s", path)

		expectedAbs[abs] = path
	}

	for _, path := range actual {
		abs, err := filepath.Abs(path)
		require.NoErrorf(tb, err, "must absolute path %s", path)

		relPath, exists := expectedAbs[abs]
		if !exists {
			continue
		}

		idx := slices.Index(expected, relPath)
		if idx >= 0 {
			expected = slices.Delete(expected, idx, idx+1)
		}
	}

	return assert.Empty(tb, expected)
}

func MustAbsPath(tb testing.TB, path string) string {
	tb.Helper()

	abs, err := filepath.Abs(path)
	require.NoError(tb, err)

	return abs
}
