// SPDX-FileCopyrightText: 2026 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package pidone_test

import (
	"bytes"
	"debug/elf"
	"testing"

	"github.com/aibor/virtrun/internal/pidone"
	"github.com/aibor/virtrun/internal/sys"
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
			data, err := pidone.For(tt.arch)
			require.ErrorIs(t, err, tt.expectedErr)

			if tt.expectedErr != nil {
				return
			}

			actual, err := elf.NewFile(bytes.NewReader(data))
			require.NoError(t, err)

			assert.Equal(t, tt.expected, actual.Machine)
		})
	}
}
