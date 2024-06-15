// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

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

func TestCPIOWriterWriteDirectory(t *testing.T) {
	t.Run("works", func(t *testing.T) {
		w := initramfs.NewCPIOWriter(&bytes.Buffer{})
		err := w.WriteDirectory("test")
		assert.NoError(t, err)
	})
	t.Run("closed", func(t *testing.T) {
		w := initramfs.NewCPIOWriter(&bytes.Buffer{})
		w.Close()
		err := w.WriteDirectory("test")
		assert.ErrorContains(t, err, "write header for test:")
	})
}

func TestCPIOWriterWriteLink(t *testing.T) {
	t.Run("works", func(t *testing.T) {
		var b bytes.Buffer
		w := initramfs.NewCPIOWriter(&b)
		err := w.WriteLink("test", "target")
		require.NoError(t, err)

		r := cpio.NewReader(&b)
		h, err := r.Next()
		require.NoError(t, err)
		assert.Equal(t, "test", h.Name)
		assert.EqualValues(t, 0o777|cpio.TypeSymlink, h.Mode)
		assert.EqualValues(t, 0, h.Size)
		assert.Equal(t, "target", h.Linkname)
	})
	t.Run("closed", func(t *testing.T) {
		w := initramfs.NewCPIOWriter(&bytes.Buffer{})
		w.Close()
		err := w.WriteLink("test", "target")
		assert.ErrorContains(t, err, "write header for test:")
	})
}

func TestCPIOWriterWriteRegular(t *testing.T) {
	fileBody := make([]byte, 200)
	for idx := range fileBody {
		fileBody[idx] = byte(idx)
	}
	testFS := fstest.MapFS{
		"regular": &fstest.MapFile{Data: fileBody},
		"dir":     &fstest.MapFile{Mode: fs.ModeDir},
		"link":    &fstest.MapFile{Mode: fs.ModeSymlink},
	}

	for _, f := range []string{"dir", "link"} {
		t.Run(f, func(t *testing.T) {
			w := initramfs.NewCPIOWriter(&bytes.Buffer{})
			file, err := testFS.Open(f)
			require.NoError(t, err)
			err = w.WriteRegular("test", file, 0o755)
			assert.ErrorContains(t, err, "not a regular file")
		})
	}

	t.Run("regular", func(t *testing.T) {
		t.Run("works", func(t *testing.T) {
			var b bytes.Buffer
			w := initramfs.NewCPIOWriter(&b)

			file, err := testFS.Open("regular")
			require.NoError(t, err)
			err = w.WriteRegular("test", file, 0o755)
			require.NoError(t, err)

			r := cpio.NewReader(&b)
			h, err := r.Next()
			require.NoError(t, err)
			assert.Equal(t, "test", h.Name)
			assert.EqualValues(t, 0o755|cpio.TypeReg, h.Mode)
			assert.EqualValues(t, 200, h.Size)

			body := make([]byte, 200)
			_, err = r.Read(body)
			require.NoError(t, err)
			assert.Equal(t, fileBody, body)
		})
		t.Run("closed", func(t *testing.T) {
			w := initramfs.NewCPIOWriter(&bytes.Buffer{})
			w.Close()

			file, err := testFS.Open("regular")
			require.NoError(t, err)
			err = w.WriteRegular("test", file, 0o755)
			assert.ErrorContains(t, err, "write header for test:")
		})
	})
}
