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
	// Collect information about reaching EOF.
	var eofReached bool

	scanner := bufio.NewScanner(p.src)
	scanner.Split(func(data []byte, atEOF bool) (int, []byte, error) {
		eofReached = atEOF
		return bufio.ScanLines(data, atEOF)
	})

	for scanner.Scan() {
		data := scanner.Bytes()

		if p.fn != nil {
			data = p.fn(data)
		}

		if p.dst == nil || data == nil {
			continue
		}

		_, err := p.dst.Write(data)
		if err != nil {
			return fmt.Errorf("write data: %w", err)
		}

		if eofReached {
			break
		}

		_, err = p.dst.Write([]byte("\n"))
		if err != nil {
			return fmt.Errorf("write newline: %w", err)
		}
	}

	if scanner.Err() != nil && !errors.Is(scanner.Err(), os.ErrClosed) {
		//nolint:wrapcheck
		return scanner.Err()
	}

	return nil
}
