// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package initramfs

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestELFFileReadInterpreter(t *testing.T) {
	interpreter, err := readInterpreter("testdata/bin/main")
	require.NoError(t, err)
	assert.Equal(t, "ld-linux-x86-64.so.2", filepath.Base(interpreter))
}

func TestELFFileLdd(t *testing.T) {
	defaultSearchPath := "testdata/lib"

	tests := []struct {
		name         string
		file         string
		searchPath   string
		expectedLibs []string
		errMsg       string
	}{
		{
			name:       "direct reference",
			file:       "testdata/lib/libfunc3.so",
			searchPath: defaultSearchPath,
			expectedLibs: []string{
				"testdata/lib/libfunc1.so",
			},
		},
		{
			name:       "indirect reference",
			file:       "testdata/bin/main",
			searchPath: defaultSearchPath,
			expectedLibs: []string{
				"testdata/lib/libfunc2.so",
				"testdata/lib/libfunc3.so",
				// libfunc1.so last since it is referenced indirectly by libfunc3.so.
				"testdata/lib/libfunc1.so",
			},
		},
		{
			name:   "fails if lib not found",
			file:   "testdata/lib/libfunc3.so",
			errMsg: "ldd: exit status 127",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("LD_LIBRARY_PATH", tt.searchPath)
			// Use interpreter of binary since the library has none.
			interpreter, err := readInterpreter("testdata/bin/main")
			require.NoError(t, err)

			infos, err := ldd(interpreter, tt.file)

			if tt.errMsg != "" {
				require.ErrorContains(t, err, tt.errMsg)
				return
			}

			require.NoErrorf(t, err, "must resolve")
			assert.Equal(t, tt.expectedLibs, infos.realPaths())
		})
	}
}

func TestLdInfosParseFrom(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
		paths []string
	}{
		{
			name: "testdata",
			// $ LD_LIBRARY_PATH=internal/files/testdata/lib/ ldd internal/files/testdata/bin/main
			lines: []string{
				"	linux-vdso.so.1 (0x00007ffeb67ab000)",
				"	libfunc2.so => internal/files/testdata/lib/libfunc2.so (0x00007f772d017000)",
				"	libfunc3.so => internal/files/testdata/lib/libfunc3.so (0x00007f772d013000)",
				"	libfunc1.so => internal/files/testdata/lib/libfunc1.so (0x00007f772d00f000)",
			},
			paths: []string{
				"internal/files/testdata/lib/libfunc2.so",
				"internal/files/testdata/lib/libfunc3.so",
				"internal/files/testdata/lib/libfunc1.so",
			},
		},
		{
			name: "env",
			// $ ldd /usr/bin/env
			lines: []string{
				"	linux-vdso.so.1 (0x00007fffec7d1000)",
				"	libc.so.6 => /usr/lib/libc.so.6 (0x00007ff161040000)",
				"	/lib64/ld-linux-x86-64.so.2 => /usr/lib64/ld-linux-x86-64.so.2 (0x00007ff161257000)",
			},
			paths: []string{
				"/usr/lib/libc.so.6",
				"/usr/lib64/ld-linux-x86-64.so.2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			for _, line := range tt.lines {
				buf.WriteString(line)
				buf.WriteRune('\n')
			}

			var infos ldInfos

			infos.parseFrom(&buf)
			assert.Equal(t, tt.paths, infos.realPaths())
		})
	}
}

func TestLdInfoParseFrom(t *testing.T) {
	tests := []struct {
		name string
		line string
		path string
	}{
		{
			name: "vdso",
			line: "	linux-vdso.so.1 (0x00007fff00ddc000)",
		},
		{
			name: "regular lib",
			line: "	libfunc2.so => testdata/lib/libfunc2.so (0x00007fb8ab53b000)",
			path: "testdata/lib/libfunc2.so",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var info ldInfo

			info.parseFrom(tt.line)
			assert.Equal(t, tt.path, info.path)
		})
	}
}
