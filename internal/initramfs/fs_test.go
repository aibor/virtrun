// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package initramfs_test

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/aibor/virtrun/internal/initramfs"
	"github.com/stretchr/testify/require"
)

func TestFiles_TestFS(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		fsys := initramfs.New()

		err := fstest.TestFS(fsys)
		require.NoError(t, err)
	})

	t.Run("non-empty", func(t *testing.T) {
		sourceFS := fstest.MapFS{
			"file": &fstest.MapFile{},
		}

		fsys := initramfs.New()

		err := fsys.Mkdir("dir")
		require.NoError(t, err)

		err = fsys.MkdirAll("dir/a/b/c")
		require.NoError(t, err)

		err = fsys.Add("dir/file", func() (fs.File, error) {
			return sourceFS.Open("file")
		})
		require.NoError(t, err)

		err = fsys.Symlink("target", "dir/link")
		require.NoError(t, err)

		err = fstest.TestFS(fsys, "dir", "dir/a/b/c", "dir/file", "dir/link")
		require.NoError(t, err)
	})
}
