// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package initramfs

import (
	"io/fs"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockWriter struct {
	Path        string
	RelatedPath string
	Source      fs.File
	Mode        fs.FileMode
	Err         error
}

func (m *MockWriter) WriteRegular(path string, source fs.File, mode fs.FileMode) error {
	m.Path = path
	m.Source = source
	m.Mode = mode

	return m.Err
}

func (m *MockWriter) WriteDirectory(path string) error {
	m.Path = path

	return m.Err
}

func (m *MockWriter) WriteLink(path, target string) error {
	m.Path = path
	m.RelatedPath = target

	return m.Err
}

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
