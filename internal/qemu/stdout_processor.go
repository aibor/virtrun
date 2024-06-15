// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package qemu

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
)

// RCFmt is the format string for communicating the test results
//
// It is parsed in the qemu wrapper. Not present in the output if the test
// binary panicked.
const RCFmt = "INIT_RC: %d\n"

var (
	// ErrGuestNoRCFound is returned if no return code matching the [RCFmt] is
	// found and no other error is found.
	ErrGuestNoRCFound = errors.New("guest did not print init return code")
	// ErrGuestPanic is returned if a kernel panic occurred in the guest
	// system.
	ErrGuestPanic = errors.New("guest system panicked")
	// ErrGuestOom is returned if the guest system ran out of memory.
	ErrGuestOom = errors.New("guest system ran out of memory")
)

var (
	panicRE = regexp.MustCompile(`^\[[0-9. ]+\] Kernel panic - not syncing: `)
	oomRE   = regexp.MustCompile(`^\[[0-9. ]+\] Out of memory: `)
)

// ParseStdout processes the input until the underlying writer is closed.
func ParseStdout(input io.Reader, output io.Writer, verbose bool) (int, error) {
	var rc int

	// rcErr is unset once a return code is found.
	rcErr := ErrGuestNoRCFound

	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case oomRE.MatchString(line):
			rcErr = ErrGuestOom
		case panicRE.MatchString(line):
			rcErr = ErrGuestPanic
		case errors.Is(rcErr, ErrGuestNoRCFound):
			if _, err := fmt.Sscanf(line, RCFmt, &rc); err == nil {
				rcErr = nil
			}
		}
		if rcErr != nil || verbose {
			if _, err := fmt.Fprintln(output, line); err != nil {
				return rc, err
			}
		}
	}

	return rc, rcErr
}
