// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package qemu

import (
	"regexp"
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
// [stdoutParser.Err]. It returns a [CommandError] with Guest flag set if either
// an error is detected or the guest communicated a non zero exit code.
type stdoutParser struct {
	ExitCodeParser
	Verbose bool

	exitCodeFound bool
	exitCode      int
	err           error
}

// Parse can be used as [lineParseFunc].
func (p *stdoutParser) Parse(data []byte) []byte {
	// Parse the output. Keep going after a match has been found, so
	// the following lines are printed as well and enhance the context
	// information in case of kernel error messages.
	switch {
	case oomRE.Match(data):
		p.err = ErrGuestOom
		return data
	case panicRE.Match(data):
		p.err = ErrGuestPanic
		return data
	case !p.exitCodeFound:
		p.exitCode, p.exitCodeFound = p.ExitCodeParser(data)
	}

	// Skip line printing once the guest exit code has been found unless the
	// verbose flag is set.
	if p.exitCodeFound && !p.Verbose {
		return nil
	}

	return data
}

// GuestSuccessful returns nil if the guest ran successfully.
//
// Otherwise, it returns a [CommandError] with the guest flag set.
func (p *stdoutParser) GuestSuccessful() error {
	err := p.err

	if err == nil {
		switch {
		case !p.exitCodeFound:
			err = ErrGuestNoExitCodeFound
		case p.exitCode != 0:
			err = ErrGuestNonZeroExitCode
		default:
			return nil
		}
	}

	return &CommandError{
		Guest:    true,
		ExitCode: p.exitCode,
		Err:      err,
	}
}
