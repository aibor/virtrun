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
		"regular": &fstest.MapFile{Data: regularFileBody},
		"dir":     &fstest.MapFile{Mode: fs.ModeDir},
		"link":    &fstest.MapFile{Mode: fs.ModeSymlink},
	}

	tests := []struct {
		name         string
		run          func(w *initramfs.CPIOWriter) error
		expectedErr  error
		assertHeader func(t assert.TestingT, hdr *cpio.Header)
		expectedBody []byte
	}{
		{
			name: "write directory",
			run: func(w *initramfs.CPIOWriter) error {
				return w.WriteDirectory("test")
			},
			assertHeader: func(t assert.TestingT, hdr *cpio.Header) {
				assert.Equal(t, "test", hdr.Name, "name")
				assert.EqualValues(t, 0o777|cpio.TypeDir, hdr.Mode, "mode")
				assert.EqualValues(t, 0, hdr.Size, "size")
			},
		},
		{
			name: "write link",
			run: func(w *initramfs.CPIOWriter) error {
				return w.WriteLink("test", "target")
			},
			assertHeader: func(t assert.TestingT, hdr *cpio.Header) {
				assert.Equal(t, "test", hdr.Name, "name")
				assert.EqualValues(t, 0o777|cpio.TypeSymlink, hdr.Mode, "mode")
				assert.EqualValues(t, 0, hdr.Size, "size")
				assert.Equal(t, "target", hdr.Linkname)
			},
		},
		{
			name: "write regular",
			run: func(w *initramfs.CPIOWriter) error {
				file, err := testFS.Open("regular")
				require.NoError(t, err)

				return w.WriteRegular("test", file, 0o755)
			},
			assertHeader: func(t assert.TestingT, hdr *cpio.Header) {
				assert.Equal(t, "test", hdr.Name, "name")
				assert.EqualValues(t, 0o755|cpio.TypeReg, hdr.Mode, "mode")
				assert.EqualValues(t, 200, hdr.Size, "size")
			},
			expectedBody: regularFileBody,
		},
		{
			name: "write regular invalid",
			run: func(w *initramfs.CPIOWriter) error {
				file, err := testFS.Open("link")
				require.NoError(t, err)

				return w.WriteRegular("test", file, 0o755)
			},
			expectedErr: initramfs.ErrNotRegularFile,
		},

		{
			name: "write closed",
			run: func(w *initramfs.CPIOWriter) error {
				err := w.Close()
				require.NoError(t, err)

				return w.WriteLink("test", "target")
			},
			expectedErr: cpio.ErrWriteAfterClose,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var archive bytes.Buffer

			w := initramfs.NewCPIOWriter(&archive)

			err := tt.run(w)
			require.ErrorIs(t, err, tt.expectedErr)

			r := cpio.NewReader(&archive)

			if tt.assertHeader == nil {
				return
			}

			h, err := r.Next()
			require.NoError(t, err)

			tt.assertHeader(t, h)

			if tt.expectedBody == nil {
				return
			}

			body := make([]byte, h.Size)
			_, err = r.Read(body)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedBody, body)
		})
	}
}
