// SPDX-FileCopyrightText: 2026 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package initramfs_test

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/aibor/virtrun/internal/initramfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
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
	actualFS, err := initramfs.New(t.Context(), spec)
	require.NoError(t, err)

	type entry struct {
		name string
		typ  fs.FileMode
	}

	actual := []entry{}

	err = fs.WalkDir(actualFS, "", func(
		path string,
		d fs.DirEntry,
		err error,
	) error {
		actual = append(actual, entry{
			name: path,
			typ:  d.Type(),
		})

		return err
	})
	require.NoError(t, err)

	expected := []entry{
		{"", fs.ModeDir},
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
