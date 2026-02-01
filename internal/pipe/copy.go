// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package pipe

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// CopyFunc defines a function that reads the data from the given reader into
// the given writer.
//
// It may copy the data as is, like io.Copy], or mutate or filter it as needed.
type CopyFunc func(dst io.Writer, src io.Reader) (int64, error)

var _ CopyFunc = io.Copy

// Decoder returns a new streaming decoder.
func Decoder(reader io.Reader) io.Reader {
	return base64.NewDecoder(base64.StdEncoding, reader)
}

var _ CopyFunc = Decode

// Decode is a [CopyFunc] that copies encoded data from src decoded to dst.
func Decode(dst io.Writer, src io.Reader) (int64, error) {
	decoder := Decoder(src)
	return io.Copy(dst, decoder) //nolint:wrapcheck
}

// DecodeLineBuffered is a [CopyFunc] that decodes and copies the data line
// buffered.
//
// This function should be used if the output is consumed line based, e.g. by
// a text parser.
func DecodeLineBuffered(dst io.Writer, src io.Reader) (int64, error) {
	var read int64

	buf := bufio.NewReader(Decoder(src))

	for {
		line, readErr := buf.ReadSlice('\n')
		read += int64(len(line))

		if len(line) > 0 {
			_, writeErr := dst.Write(line)
			if writeErr != nil {
				return read, fmt.Errorf("write: %w", writeErr)
			}
		}

		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return read, nil
			}

			return read, fmt.Errorf("read: %w", readErr)
		}
	}
}

// Encoder returns a new streaming encoder.
//
// The host side is expected to use [Decoder] for decoding the console
// output.
func Encoder(written io.Writer) io.WriteCloser {
	return base64.NewEncoder(base64.StdEncoding, written)
}

var _ CopyFunc = Encode

// Encode is a [CopyFunc] that copies plain data read from src encoded to dst.
func Encode(dst io.Writer, src io.Reader) (int64, error) {
	encoder := Encoder(dst)
	defer encoder.Close()

	return io.Copy(encoder, src) //nolint:wrapcheck
}
