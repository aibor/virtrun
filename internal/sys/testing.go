// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sys

import (
	"path/filepath"
	"slices"
	"testing"
)

func AssertContainsPaths(tb testing.TB, actual, expected []string) bool {
	tb.Helper()

	expectedAbs := make(map[string]string, len(expected))

	for _, path := range expected {
		abs := MustAbsPath(tb, path)

		expectedAbs[abs] = path
	}

	for _, path := range actual {
		abs := MustAbsPath(tb, path)

		relPath, exists := expectedAbs[abs]
		if !exists {
			continue
		}

		idx := slices.Index(expected, relPath)
		if idx >= 0 {
			expected = slices.Delete(expected, idx, idx+1)
		}
	}

	if len(expected) > 0 {
		tb.Errorf("expected paths not present: % s", expected)
		return false
	}

	return true
}

func MustAbsPath(tb testing.TB, path string) string {
	tb.Helper()

	abs, err := filepath.Abs(path)
	if err != nil {
		tb.Fatalf("failed to get absolute path %s: %v", path, err)
	}

	return abs
}
