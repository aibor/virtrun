// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package initramfs_test

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/aibor/virtrun/internal/initramfs"
	"github.com/cavaliergopher/cpio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCPIOFSWriter_AddFS(t *testing.T) {
	sourceFS := fstest.MapFS{
		".": &fstest.MapFile{
			Mode: fs.ModeDir,
		},
		"regular": &fstest.MapFile{
			Data: slices.Repeat([]byte{0xfe}, 200),
		},
		"link": &fstest.MapFile{
			Data: []byte("target"),
			Mode: fs.ModeSymlink,
		},
		"dir": &fstest.MapFile{
			Mode: fs.ModeDir,
		},
		"dir/regular": &fstest.MapFile{
			Data: slices.Repeat([]byte{0xfe}, 100),
		},
		"dir/link": &fstest.MapFile{
			Data: []byte("/"),
			Mode: fs.ModeSymlink,
		},
	}

	var archive bytes.Buffer

	w := initramfs.NewCPIOFSWriter(&archive)

	err := w.AddFS(initramfs.WithReadLinkNoFollowOpen(sourceFS))
	require.NoError(t, err)

	r := cpio.NewReader(&archive)

	extractedFS := fstest.MapFS{}

	for {
		hdr, err := r.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		require.NoError(t, err)

		var body []byte
		if hdr.Size > 0 {
			body = make([]byte, hdr.Size)
			_, err = r.Read(body)
			require.NoError(t, err)
		} else if hdr.Linkname != "" {
			body = []byte(hdr.Linkname)
		}

		extractedFS[hdr.Name] = &fstest.MapFile{
			Data: body,
			Mode: hdr.FileInfo().Mode(),
		}
	}

	assert.Equal(t, sourceFS, extractedFS)
}
