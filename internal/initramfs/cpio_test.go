// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package initramfs_test

import (
	"bytes"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/aibor/virtrun/internal/initramfs"
	"github.com/cavaliergopher/cpio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCPIOWriter(t *testing.T) {
	regularFileBody := make([]byte, 200)
	for idx := range regularFileBody {
		regularFileBody[idx] = byte(idx)
	}

	testFS := fstest.MapFS{
		"regular": &fstest.MapFile{
			Data: regularFileBody,
		},
		"dir": &fstest.MapFile{
			Mode: fs.ModeDir,
		},
		"link": &fstest.MapFile{
			Data: []byte("target"),
			Mode: fs.ModeSymlink,
		},
	}

	tests := []struct {
		name             string
		fileName         string
		expectedErr      error
		expectedType     uint
		expectedSize     int64
		expectedLinks    int
		expectedLinkname string
		expectedBody     []byte
	}{
		{
			name:          "write directory",
			fileName:      "dir",
			expectedType:  cpio.TypeDir,
			expectedLinks: 2,
		},
		{
			name:             "write link",
			fileName:         "link",
			expectedType:     cpio.TypeSymlink,
			expectedLinkname: "target",
		},
		{
			name:          "write regular",
			fileName:      "regular",
			expectedType:  cpio.TypeReg,
			expectedSize:  200,
			expectedLinks: 1,
			expectedBody:  regularFileBody,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var archive bytes.Buffer

			w := initramfs.NewCPIOFileWriter(&archive)

			file, err := testFS.Open(tt.fileName)
			require.NoError(t, err)

			err = w.WriteFile("test", file)
			require.ErrorIs(t, err, tt.expectedErr)

			if tt.expectedErr != nil {
				return
			}

			r := cpio.NewReader(&archive)

			hdr, err := r.Next()
			require.NoError(t, err)

			assert.Equal(t, "test", hdr.Name, "name")
			assert.EqualValues(t, tt.expectedType, hdr.Mode, "mode")
			assert.EqualValues(t, tt.expectedSize, hdr.Size, "size")
			assert.EqualValues(t, tt.expectedLinks, hdr.Links, "links")

			if tt.expectedBody == nil {
				return
			}

			body := make([]byte, hdr.Size)
			_, err = r.Read(body)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedBody, body)
		})
	}
}
