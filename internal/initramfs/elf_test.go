// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package initramfs_test

import (
	"testing"

	"github.com/aibor/virtrun/internal/initramfs"
	"github.com/stretchr/testify/assert"
)

func TestFilesLdd(t *testing.T) {
	defaultSearchPath := "testdata/lib"

	tests := []struct {
		name          string
		file          string
		searchPath    string
		expectedPaths []string
		errMsg        string
	}{
		{
			name:       "indirect reference",
			file:       "testdata/bin/main",
			searchPath: defaultSearchPath,
			expectedPaths: []string{
				"testdata/lib/libfunc2.so",
				"testdata/lib/libfunc3.so",
				// libfunc1.so last since it is referenced indirectly by libfunc3.so.
				"testdata/lib/libfunc1.so",
				"/lib64/ld-linux-x86-64.so.2",
			},
		},
		{
			name:   "fails if lib not found",
			file:   "testdata/bin/main",
			errMsg: "ldd: exit status 127: testdata/bin/main",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("LD_LIBRARY_PATH", tt.searchPath)
			paths, err := initramfs.Ldd(tt.file)
			if tt.errMsg == "" {
				assert.NoErrorf(t, err, "must resolve")
			} else {
				assert.ErrorContains(t, err, tt.errMsg)
			}

			assert.Equal(t, tt.expectedPaths, paths)
		})
	}
}
