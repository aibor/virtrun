// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package virtrun

import (
	"io"
	"testing"

	"github.com/aibor/virtrun/internal/sys"
	"github.com/stretchr/testify/require"
)

func TestValidateELF(t *testing.T) {
	tests := []struct {
		name        string
		arch        sys.Arch
		fileForArch sys.Arch
		expectedErr error
	}{
		{
			name:        "valid amd64",
			arch:        sys.AMD64,
			fileForArch: sys.AMD64,
		},
		{
			name:        "valid arm64",
			arch:        sys.ARM64,
			fileForArch: sys.ARM64,
		},
		{
			name:        "valid riscv64",
			arch:        sys.RISCV64,
			fileForArch: sys.RISCV64,
		},
		{
			name:        "invalid arch",
			arch:        sys.RISCV64,
			fileForArch: sys.AMD64,
			expectedErr: ErrMachineNotSupported,
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.arch), func(t *testing.T) {
			file, err := initProgFor(tt.fileForArch)
			require.NoError(t, err)

			t.Cleanup(func() { _ = file.Close() })

			seekFile, ok := file.(io.ReaderAt)
			require.True(t, ok, "must implement io.ReaderAt")

			err = ValidateELF(seekFile, tt.arch)
			require.ErrorIs(t, err, tt.expectedErr)
		})
	}
}
