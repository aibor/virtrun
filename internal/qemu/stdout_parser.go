// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package qemu

import (
	"bufio"
	"fmt"
	"io"
	"regexp"

	"github.com/aibor/virtrun/internal/pipe"
)

var (
	panicRE = regexp.MustCompile(`^\[[0-9. ]+\] Kernel panic - not syncing: `)
	oomRE   = regexp.MustCompile(`^\[[0-9. ]+\] Out of memory: `)
)

// ExitCodeParser parses the given string and returns the exit code found in
// the input or an error if the input does not contain the expected string or
// parsing fails otherwise.
type ExitCodeParser func(line []byte) (int, bool)

// stdoutParser provides a parser that parses stdout from the guest.
//
// It detects kernel panics, OOM messages and most importantly it detects the
// exit code communicated by the guest via stdout. The processor stops when
// the src is closed. After use, the result can be retrieved by calling
// [Parser.Err]. It returns a [CommandError] with Guest flag set if either
// an error is detected or the guest communicated a non zero exit code.
type stdoutParser struct {
	ExitCodeParser

	Verbose bool

	exitCodeFound bool
	exitCode      int
	err           error
}

var _ pipe.CopyFunc = (*stdoutParser)(nil).Copy

// Copy reads from the given reader which is expected to be output from a
// guest's standard output.
//
// Once done [stdoutParser.Result] can be called to retrieve the result.
func (p *stdoutParser) Copy(dst io.Writer, src io.Reader) (int64, error) {
	var read int64

	scanner := bufio.NewScanner(src)
	scanner.Split(func(data []byte, atEOF bool) (int, []byte, error) {
		advance, token, err := bufio.ScanLines(data, atEOF)
		read += int64(advance)

		return advance, token, err //nolint:wrapcheck
	})

	for scanner.Scan() {
		data := scanner.Bytes()
		if !p.parseLine(data) {
			continue
		}

		data = append(data, '\n')

		_, err := dst.Write(data)
		if err != nil {
			return read, fmt.Errorf("print: %w", err)
		}
	}

	if p.err == nil && !p.exitCodeFound {
		p.err = ErrGuestNoExitCodeFound
	}

	return read, scanner.Err() //nolint:wrapcheck
}

// Result returns the exitcode and any error found in the guest's output.
//
// It is only valid to call after [stdoutParser.Copy] ran and terminated
// successfully.
func (p *stdoutParser) Result() (int, error) {
	return p.exitCode, p.err
}

// Parse can be used as [lineParseFunc].
func (p *stdoutParser) parseLine(data []byte) bool {
	// Parse the output. Keep going after a match has been found, so
	// the following lines are printed as well and enhance the context
	// information in case of kernel error messages.
	switch {
	case oomRE.Match(data):
		p.err = ErrGuestOom
		return true
	case panicRE.Match(data):
		p.err = ErrGuestPanic
		return true
	case !p.exitCodeFound:
		p.exitCode, p.exitCodeFound = p.ExitCodeParser(data)
		if p.exitCode != 0 {
			p.err = ErrGuestNonZeroExitCode
		}
	}

	// Skip line printing once the guest exit code has been found unless the
	// verbose flag is set.
	return !p.exitCodeFound || p.Verbose
}
