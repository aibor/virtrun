// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package pipe_test

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/aibor/virtrun/internal/pipe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type errWriter struct{}

func (errWriter) Write(_ []byte) (int, error) {
	return 0, assert.AnError
}

func (errWriter) String() string {
	return ""
}

func TestDecodeLineBuffered(t *testing.T) {
	encoder := func(data string) io.Reader {
		return strings.NewReader(
			base64.StdEncoding.EncodeToString([]byte(data)),
		)
	}

	tests := []struct {
		name   string
		reader io.Reader
		writer interface {
			io.Writer
			fmt.Stringer
		}
		expected    string
		expectedN   int64
		expectedErr error
	}{
		{
			name:   "read eof",
			reader: iotest.ErrReader(io.EOF),
			writer: &bytes.Buffer{},
		},
		{
			name:      "read data with eof",
			reader:    iotest.DataErrReader(encoder("test\ndata\nmore")),
			writer:    &bytes.Buffer{},
			expected:  "test\ndata\nmore\n",
			expectedN: 15,
		},
		{
			// bufio.Reader has 4096 bytes buffer.
			name:      "read data with full buf",
			reader:    encoder(strings.Repeat("testdata", 4096) + "\n"),
			writer:    &bytes.Buffer{},
			expected:  strings.Repeat("testdata", 4096) + "\n",
			expectedN: 8*4096 + 1,
		},
		{
			name:        "read error",
			reader:      iotest.TimeoutReader(encoder("test\ndata\n")),
			writer:      &bytes.Buffer{},
			expected:    "test\ndata\n",
			expectedN:   10,
			expectedErr: iotest.ErrTimeout,
		},
		{
			name:        "write error",
			reader:      encoder("test\ndata\n"),
			writer:      errWriter{},
			expectedN:   0,
			expectedErr: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := pipe.DecodeLineBuffered(tt.writer, tt.reader)
			require.ErrorIs(t, err, tt.expectedErr)
			assert.Equal(t, tt.expectedN, n)
			assert.Equal(t, tt.expected, tt.writer.String())
		})
	}
}

func TestEncoderDecoder(t *testing.T) {
	var output, encoded bytes.Buffer

	input := "test string\nwith another line\r\n"

	encoder := pipe.Encoder(&encoded)
	_, err := io.Copy(encoder, strings.NewReader(input))
	require.NoError(t, err)

	err = encoder.Close()
	require.NoError(t, err)

	decoder := pipe.Decoder(&encoded)
	_, err = io.Copy(&output, decoder)
	require.NoError(t, err)

	assert.Equal(t, input, output.String())
}

func TestEncodeDecode(t *testing.T) {
	var output, encoded bytes.Buffer

	input := "test string\nwith another line\r\n"

	_, err := pipe.Encode(&encoded, strings.NewReader(input))
	require.NoError(t, err)

	_, err = pipe.Decode(&output, &encoded)
	require.NoError(t, err)

	assert.Equal(t, input, output.String())
}
