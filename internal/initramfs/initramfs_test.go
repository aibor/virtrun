// SPDX-FileCopyrightText: 2026 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package initramfs_test

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"testing"
	"testing/fstest"

	"github.com/aibor/cpio"
	"github.com/aibor/virtrun/internal/initramfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitramfs(t *testing.T) {
	testFS := fstest.MapFS{
		"binary":           {Data: []byte("binary")},
		"some/file1":       {Data: []byte("file 1")},
		"other/file2":      {Data: []byte("file 2")},
		"modules/a.ko.zst": {Data: []byte("module a")},
		"modules/b.ko.zst": {Data: []byte("module b")},
		"init":             {Data: []byte("init")},
	}

	init, err := testFS.Open("init")
	require.NoError(t, err)

	spec := initramfs.Spec{
		Executable: "binary",
		Files: []string{
			"some/file1",
			"other/file2",
		},
		Modules: []string{
			"modules/b.ko.zst",
			"modules/a.ko.zst",
		},
		Fsys: testFS,
		Init: init,
	}

	t.Setenv("TMPDIR", t.TempDir())
	path, err := initramfs.BuildArchive(t.Context(), spec)
	require.NoError(t, err)

	archive, err := os.Open(path)
	require.NoError(t, err)

	type entry struct {
		name string
		typ  fs.FileMode
	}

	cpioReader := cpio.NewReader(archive)
	actual := []entry{}

	for {
		hdr, err := cpioReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		require.NoError(t, err)

		actual = append(actual, entry{
			name: hdr.Name,
			typ:  hdr.FileInfo().Mode().Type(),
		})
	}

	expected := []entry{
		{"data", fs.ModeDir},
		{"data/file1", 0},
		{"data/file2", 0},
		{"init", 0},
		{"lib", fs.ModeDir},
		{"lib/modules", fs.ModeDir},
		{"lib/modules/0000-b.ko.zst", 0},
		{"lib/modules/0001-a.ko.zst", 0},
		{"main", 0},
		{"run", fs.ModeDir},
		{"tmp", fs.ModeDir},
	}

	assert.Equal(t, expected, actual)
}
