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
