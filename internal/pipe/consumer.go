// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package pipe

import (
	"encoding/base64"
	"fmt"
	"io"
)

type Consumer interface {
	io.ReaderFrom
	io.Closer
}

const pathPrefix = "/host_pipe"

// Path creates the absolute host pipe path for the given port.
func Path(idx int) string {
	return fmt.Sprintf("%s%d", pathPrefix, idx)
}

type decoder struct {
	io.WriteCloser
}

func (e *decoder) ReadFrom(reader io.Reader) (int64, error) {
	decoder := base64.NewDecoder(base64.StdEncoding, reader)
	return io.Copy(e.WriteCloser, decoder) //nolint:wrapcheck
}

// Decoder returns a new streaming decoder.
func Decoder(writer io.WriteCloser) Consumer { //nolint:ireturn
	return &decoder{writer}
}

type encoder struct {
	io.WriteCloser
}

func (e *encoder) ReadFrom(reader io.Reader) (int64, error) {
	encoder := base64.NewEncoder(base64.StdEncoding, e.WriteCloser)
	defer encoder.Close()

	return io.Copy(encoder, reader) //nolint:wrapcheck
}

// Encoder returns a new streaming encoder.
//
// The host side is expected to use [Decoder] for decoding the console
// output.
func Encoder(writer io.WriteCloser) Consumer { //nolint:ireturn
	return &encoder{writer}
}
