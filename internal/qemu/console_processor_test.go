package qemu_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/aibor/virtrun/internal/qemu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testWriteCloser struct {
	bytes.Buffer
}

func (t testWriteCloser) Close() error {
	return nil
}

func TestConsoleProcessor_Run(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectedErr error
	}{
		{
			name: "empty",
		},
		{
			name:     "crlf only",
			input:    "\r\n",
			expected: "\n",
		},
		{
			name:     "lf only",
			input:    "\n",
			expected: "\n",
		},
		{
			name:     "with crlf",
			input:    "some first\r\nand second\r\nand third line",
			expected: "some first\nand second\nand third line\n",
		},
		{
			name:     "with lf",
			input:    "some first\nand second\nand third line",
			expected: "some first\nand second\nand third line\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output testWriteCloser

			processor, err := qemu.NewConsoleProcessor(&output)
			require.NoError(t, err)

			go func() {
				io.Copy(processor.WritePipe, bytes.NewBuffer([]byte(tt.input)))
				processor.WritePipe.Close()
			}()

			err = processor.Run()
			require.ErrorIs(t, err, tt.expectedErr)

			assert.Equal(t, tt.expected, output.String())
		})
	}
}
