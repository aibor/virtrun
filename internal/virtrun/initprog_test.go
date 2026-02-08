// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package virtrun_test

import (
	"debug/elf"
	"io"
	"testing"

	"github.com/aibor/virtrun/internal/sys"
	"github.com/aibor/virtrun/internal/virtrun"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInits(t *testing.T) {
	tests := []struct {
		name        string
		arch        sys.Arch
		expected    elf.Machine
		expectedErr error
	}{
		{
			name:     "valid amd64",
			arch:     sys.AMD64,
			expected: elf.EM_X86_64,
		},
		{
			name:     "valid arm64",
			arch:     sys.ARM64,
			expected: elf.EM_AARCH64,
		},
		{
			name:     "valid riscv64",
			arch:     sys.RISCV64,
			expected: elf.EM_RISCV,
		},
		{
			name:        "unknown arch",
			arch:        "mips64",
			expectedErr: sys.ErrArchNotSupported,
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.arch), func(t *testing.T) {
			file, err := virtrun.InitProgFor(tt.arch)
			require.ErrorIs(t, err, tt.expectedErr)

			if tt.expectedErr != nil {
				return
			}

			readerAt, ok := file.(io.ReaderAt)
			require.True(t, ok, "file must implement io.ReaderAt")

			actual, err := elf.NewFile(readerAt)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, actual.Machine)
		})
	}
}
