// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package initramfs

import (
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFS_Add(t *testing.T) {
	fsys := New()

	err := fsys.Mkdir("dir")
	require.NoError(t, err)

	expected := map[string]string{
		"second": "rel/third",
		"fourth": "/abs/fourth",
	}

	for name := range expected {
		require.NoError(t, fsys.Add("dir/"+name, func() (fs.File, error) {
			return &openFile{}, nil
		}))
	}

	for name := range expected {
		path := filepath.Join("dir", name)

		e, err := fsys.find(path)
		require.NoError(t, err, path)

		_, ok := e.(regularFile)
		require.True(t, ok)
	}
}

func TestFS_Mkdir(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		prepare     func(fsys *FS) error
		expectedErr error
	}{
		{
			name: "new",
			path: "dir",
		},
		{
			name: "new in dir",
			path: "dir/sub",
			prepare: func(fsys *FS) error {
				return fsys.Mkdir("dir")
			},
		},
		{
			name: "exists as dir",
			path: "dir",
			prepare: func(fsys *FS) error {
				return fsys.Mkdir("dir")
			},
			expectedErr: ErrFileExist,
		},
		{
			name: "exists as other",
			path: "dir",
			prepare: func(fsys *FS) error {
				return fsys.Symlink("somewhere", "dir")
			},
			expectedErr: ErrFileExist,
		},
		{
			name: "parent not a dir",
			path: "dir/sub",
			prepare: func(fsys *FS) error {
				return fsys.Symlink("somewhere", "dir")
			},
			expectedErr: ErrFileNotDir,
		},
		{
			name:        "missing parent",
			path:        "dir/sub",
			expectedErr: ErrFileNotExist,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := New()

			if tt.prepare != nil {
				err := tt.prepare(fsys)
				require.NoError(t, err)
			}

			err := fsys.Mkdir(tt.path)
			require.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

func TestFS_MkdirAll(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		prepare     func(fsys *FS) error
		expectedErr error
	}{
		{
			name: "new",
			path: "dir",
		},
		{
			name: "new in dir",
			path: "dir/sub",
			prepare: func(fsys *FS) error {
				return fsys.Mkdir("dir")
			},
		},
		{
			name: "exists as dir",
			path: "dir",
			prepare: func(fsys *FS) error {
				return fsys.Mkdir("dir")
			},
		},
		{
			name: "exists as other",
			path: "dir",
			prepare: func(fsys *FS) error {
				return fsys.Symlink("somewhere", "dir")
			},
			expectedErr: ErrFileNotDir,
		},
		{
			name: "parent not a dir",
			path: "dir/sub/subsub",
			prepare: func(fsys *FS) error {
				return fsys.Symlink("somewhere", "dir")
			},
			expectedErr: ErrFileNotDir,
		},
		{
			name: "missing parent",
			path: "dir/sub",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := New()

			if tt.prepare != nil {
				err := tt.prepare(fsys)
				require.NoError(t, err)
			}

			err := fsys.MkdirAll(tt.path)
			require.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

func TestFS_Symlink(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		prepare     func(fsys *FS) error
		expectedErr error
	}{
		{
			name: "new",
			path: "link",
		},
		{
			name: "new in dir",
			path: "dir/link",
			prepare: func(fsys *FS) error {
				return fsys.Mkdir("dir")
			},
		},
		{
			name: "exists as link",
			prepare: func(fsys *FS) error {
				return fsys.Symlink("somewhere", "link")
			},
			path:        "link",
			expectedErr: ErrFileExist,
		},
		{
			name: "exists as other",
			path: "link",
			prepare: func(fsys *FS) error {
				return fsys.Mkdir("link")
			},
			expectedErr: ErrFileExist,
		},
		{
			name: "parent not a dir",
			path: "dir/link",
			prepare: func(fsys *FS) error {
				return fsys.Symlink("somewhere", "dir")
			},
			expectedErr: ErrFileNotDir,
		},
		{
			name:        "missing parent",
			path:        "dir/link",
			expectedErr: ErrFileNotExist,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := New()

			if tt.prepare != nil {
				err := tt.prepare(fsys)
				require.NoError(t, err)
			}

			err := fsys.Symlink("somewhere", tt.path)
			require.ErrorIs(t, err, tt.expectedErr)
		})
	}
}
