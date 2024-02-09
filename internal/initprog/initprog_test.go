package initprog_test

import (
	"debug/elf"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aibor/virtrun/internal/initprog"
)

func TestInits(t *testing.T) {
	tests := []struct {
		arch    string
		machine elf.Machine
		errMsg  string
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
			arch:   "unsupported",
			errMsg: "arch not supported",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.arch, func(t *testing.T) {
			file, err := initprog.For(tt.arch)
			if tt.errMsg != "" {
				assert.ErrorContains(t, err, tt.errMsg)
				return
			}
			require.NoError(t, err)
			t.Cleanup(func() { _ = file.Close() })

			seekFile, ok := file.(io.ReaderAt)
			if !ok {
				t.Skip("fs.File does not implement io.ReaderAt yet")
			}

			elfFile, err := elf.NewFile(seekFile)
			require.NoError(t, err)

			assert.Equal(t, elf.ELFOSABI_NONE, elfFile.OSABI)
			assert.Equal(t, tt.machine, elfFile.Machine)
		})
	}
}
