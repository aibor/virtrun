// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package pipe_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/aibor/virtrun/internal/pipe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
