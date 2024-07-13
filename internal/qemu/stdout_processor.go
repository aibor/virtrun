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
const RCFmt = "INIT_RC: %d"

var (
	panicRE = regexp.MustCompile(`^\[[0-9. ]+\] Kernel panic - not syncing: `)
	oomRE   = regexp.MustCompile(`^\[[0-9. ]+\] Out of memory: `)
)

// ParseStdout processes the input until the underlying writer is closed.
func ParseStdout(input io.Reader, output io.Writer, verbose bool) (int, error) {
	var exitCode int

	// guestErr is unset once an exit code is found in the output stream.
	guestErr := ErrGuestNoExitCodeFound

	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()

		// Parse the output. Keep going after a match has been found, so
		// the following lines are printed as well and enhance the context
		// information in case of kernel error messages.
		switch {
		case oomRE.MatchString(line):
			guestErr = ErrGuestOom
		case panicRE.MatchString(line):
			guestErr = ErrGuestPanic
		case errors.Is(guestErr, ErrGuestNoExitCodeFound):
			_, err := fmt.Sscanf(line, RCFmt, &exitCode)
			if err != nil {
				break
			}

			guestErr = nil
		}

		// Skip line printing once the init exit code has been found unless
		// the verbose flag is set.
		if guestErr == nil && !verbose {
			continue
		}

		_, err := fmt.Fprintln(output, line)
		if err != nil {
			return exitCode, fmt.Errorf("print: %w", err)
		}
	}

	return exitCode, guestErr
}
