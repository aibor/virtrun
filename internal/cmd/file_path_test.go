// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd_test

import (
	"testing"

	"github.com/aibor/virtrun/internal/cmd"
	"github.com/aibor/virtrun/internal/sys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilePath_Set(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectedErr error
	}{
		{
			name:        "empty",
			expectedErr: sys.ErrEmptyPath,
		},
		{
			name:     "valid",
			input:    "path",
			expected: sys.MustAbsolutePath("path"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var path cmd.FilePath

			err := path.Set(tt.input)
			require.ErrorIs(t, err, tt.expectedErr)

			assert.Equal(t, tt.expected, string(path))
		})
	}
}

func TestFilePath_String(t *testing.T) {
	path := cmd.FilePath("/path")
	assert.Equal(t, "/path", path.String())
}

func TestFilePathList_Set(t *testing.T) {
	tests := []struct {
		name     string
		list     cmd.FilePathList
		inputs   []string
		expected cmd.FilePathList
	}{
		{
			name: "single",
			inputs: []string{
				"path",
			},
			expected: cmd.FilePathList{
				sys.MustAbsolutePath("path"),
			},
		},
		{
			name: "multi",
			inputs: []string{
				"/path",
				"otherpath",
				"third",
			},
			expected: cmd.FilePathList{
				"/path",
				sys.MustAbsolutePath("otherpath"),
				sys.MustAbsolutePath("third"),
			},
		},
		{
			name: "comma",
			inputs: []string{
				"/path,otherpath,third",
			},
			expected: cmd.FilePathList{
				"/path",
				sys.MustAbsolutePath("otherpath"),
				sys.MustAbsolutePath("third"),
			},
		},
		{
			name: "add",
			list: cmd.FilePathList{
				"/path",
				sys.MustAbsolutePath("otherpath"),
			},
			inputs: []string{
				"third",
			},
			expected: cmd.FilePathList{
				"/path",
				sys.MustAbsolutePath("otherpath"),
				sys.MustAbsolutePath("third"),
			},
		},
		{
			name: "reset",
			list: cmd.FilePathList{
				"/path",
				sys.MustAbsolutePath("otherpath"),
			},
			inputs: []string{
				"",
				"third",
			},
			expected: cmd.FilePathList{
				sys.MustAbsolutePath("third"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, input := range tt.inputs {
				require.NoError(t, tt.list.Set(input))
			}

			assert.Equal(t, tt.expected, tt.list)
		})
	}
}

func TestFilePathList_String(t *testing.T) {
	tests := []struct {
		name     string
		list     cmd.FilePathList
		expected string
	}{
		{
			name: "empty",
		},
		{
			name: "single",
			list: cmd.FilePathList{
				"/path",
			},
			expected: "/path",
		},
		{
			name: "multi",
			list: cmd.FilePathList{
				"/path",
				"/otherpath",
				"/third",
			},
			expected: "/path,/otherpath,/third",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.list.String()
			assert.Equal(t, tt.expected, actual)
		})
	}
}
