// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package virtrun

import (
	"debug/elf"
	"io"
	"testing"

	"github.com/aibor/virtrun/internal/sys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInits(t *testing.T) {
	tests := []struct {
		arch        sys.Arch
		machine     elf.Machine
		expectedErr error
	}{
		{
			arch:    "amd64",
			machine: elf.EM_X86_64,
		},
		{
			arch:    "arm64",
			machine: elf.EM_AARCH64,
		},
		{
			arch:        "unsupported",
			expectedErr: sys.ErrArchNotSupported,
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.arch), func(t *testing.T) {
			file, err := initProgFor(tt.arch)
			if file != nil {
				t.Cleanup(func() { _ = file.Close() })
			}

			require.ErrorIs(t, err, tt.expectedErr)

			if tt.expectedErr != nil {
				return
			}

			seekFile, ok := file.(io.ReaderAt)
			require.True(t, ok, "must implement io.ReaderAt")

			elfFile, err := elf.NewFile(seekFile)
			require.NoError(t, err)

			assert.Equal(t, elf.ELFOSABI_NONE, elfFile.OSABI)
			assert.Equal(t, tt.machine, elfFile.Machine)
		})
	}
}
