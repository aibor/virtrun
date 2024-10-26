// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package virtrun

import (
	"testing"

	"github.com/aibor/virtrun/internal/sys"
	"github.com/stretchr/testify/require"
)

func TestInits(t *testing.T) {
	tests := []struct {
		name        string
		arch        sys.Arch
		expectedErr error
	}{
		{
			name: "valid amd64",
			arch: sys.AMD64,
		},
		{
			name: "valid arm64",
			arch: sys.ARM64,
		},
		{
			name: "valid riscv64",
			arch: sys.RISCV64,
		},
		{
			name:        "unknown arch",
			arch:        "mips64",
			expectedErr: sys.ErrArchNotSupported,
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.arch), func(t *testing.T) {
			_, err := initProgFor(tt.arch)
			require.ErrorIs(t, err, tt.expectedErr)
		})
	}
}
