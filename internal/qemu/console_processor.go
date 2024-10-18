// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package qemu

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
)

type lineParseFunc func([]byte) []byte

// consoleProcessor is a generic processor of serial console output.
//
// For each line read from src the given [lineParseFunc] is called. If the
// function returns non-nil data and dst is set, the output is written to dst.
//
// It can be used without a parse function set to just sanitize line endings.
type consoleProcessor struct {
	dst io.Writer
	src io.Reader
	fn  lineParseFunc
}

func (p consoleProcessor) run() error {
	scanner := bufio.NewScanner(p.src)
	for scanner.Scan() {
		data := scanner.Bytes()

		if p.fn != nil {
			data = p.fn(data)
		}

		err := p.writeLn(data)
		if err != nil {
			return err
		}
	}

	if scanner.Err() != nil && !errors.Is(scanner.Err(), os.ErrClosed) {
		//nolint:wrapcheck
		return scanner.Err()
	}

	return nil
}

func (p consoleProcessor) writeLn(data []byte) error {
	// If the there is no output writer or the passed data is nil, discard it.
	if p.dst == nil || data == nil {
		return nil
	}

	_, err := p.dst.Write(data)
	if err != nil {
		return fmt.Errorf("write data: %w", err)
	}

	_, err = p.dst.Write([]byte("\n"))
	if err != nil {
		return fmt.Errorf("write newline: %w", err)
	}

	return nil
}
