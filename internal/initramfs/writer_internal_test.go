// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package initramfs

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestWriteFS(t *testing.T) {
	mapDirArg := mock.AnythingOfType("*fstest.mapDir")
	mapFileArg := mock.AnythingOfType("*fstest.openMapFile")
	fsFileArg := mock.AnythingOfType("*initramfs.openFile")

	errFS := New()
	err := errFS.Add("file", func() (fs.File, error) {
		return nil, assert.AnError
	})
	require.NoError(t, err)

	tests := []struct {
		name      string
		fs        fs.FS
		prepare   func(m *MockWriter)
		assertErr require.ErrorAssertionFunc
	}{
		{
			name: "succeeds",
			fs: fstest.MapFS{
				"dir/file": &fstest.MapFile{},
			},
			prepare: func(m *MockWriter) {
				m.On("WriteFile", ".", mapDirArg).
					Once().Return(nil)
				m.On("WriteFile", "dir", mapDirArg).
					Once().Return(nil)
				m.On("WriteFile", "dir/file", mapFileArg).
					Once().Return(nil)
			},
			assertErr: require.NoError,
		},
		{
			name: "open fails",
			fs:   errFS,
			prepare: func(m *MockWriter) {
				m.On("WriteFile", ".", fsFileArg).
					Once().Return(nil)
			},
			assertErr: func(t require.TestingT, err error, a ...any) {
				require.ErrorIs(t, err, assert.AnError, a...)
			},
		},
		{
			name: "write fails",
			fs:   fstest.MapFS{},
			prepare: func(m *MockWriter) {
				m.On("WriteFile", ".", mapDirArg).
					Once().Return(assert.AnError)
			},
			assertErr: func(t require.TestingT, err error, a ...any) {
				require.ErrorIs(t, err, assert.AnError, a...)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := MockWriter{}
			tt.prepare(&writer)

			err := WriteFS(tt.fs, &writer)
			tt.assertErr(t, err)

			writer.AssertExpectations(t)
		})
	}
}
