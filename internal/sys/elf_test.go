// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sys_test

import (
	"io/fs"
	"testing"

	"github.com/aibor/virtrun/internal/sys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadArch(t *testing.T) {
	tests := []struct {
		name      string
		expected  sys.Arch
		assertErr require.ErrorAssertionFunc
	}{
		{
			name:      "amd64",
			expected:  sys.AMD64,
			assertErr: require.NoError,
		},
		{
			name:      "arm64",
			expected:  sys.ARM64,
			assertErr: require.NoError,
		},
		{
			name:      "riscv64",
			expected:  sys.RISCV64,
			assertErr: require.NoError,
		},
		{
			name: "unknown",
			assertErr: func(t require.TestingT, err error, _ ...any) {
				require.ErrorIs(t, err, fs.ErrNotExist)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileName := "../virtrun/bin/" + tt.name
			actual, err := sys.ReadELFArch(fileName)
			tt.assertErr(t, err)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
