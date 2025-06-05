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
	dst        io.Writer
	src        io.Reader
	fn         lineParseFunc
	eofReached bool
}

func (p *consoleProcessor) run() error {
	scanner := bufio.NewScanner(p.src)
	scanner.Split(func(data []byte, atEOF bool) (int, []byte, error) {
		p.eofReached = atEOF
		return bufio.ScanLines(data, atEOF)
	})

	var called bool
	for scanner.Scan() {
		called = true

		if err := p.processLine(scanner.Bytes()); err != nil {
			return err
		}
	}

	if scanner.Err() != nil && !errors.Is(scanner.Err(), os.ErrClosed) {
		//nolint:wrapcheck
		return scanner.Err()
	}

	if !called {
		return ErrConsoleNoOutput
	}

	return nil
}

func (p *consoleProcessor) processLine(data []byte) error {
	if p.fn != nil {
		data = p.fn(data)
	}

	if p.dst == nil || data == nil {
		return nil
	}

	if _, err := p.dst.Write(data); err != nil {
		return fmt.Errorf("write data: %w", err)
	}

	if p.eofReached {
		return nil
	}

	if _, err := p.dst.Write([]byte("\n")); err != nil {
		return fmt.Errorf("write newline: %w", err)
	}

	return nil
}
