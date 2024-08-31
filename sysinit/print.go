// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"fmt"
	"io"
)

// ExitCodeFmt is the format string for communicating the test results
//
// The same format string must be configured for the [qemu.Command] so it is
// matched correctly.
const ExitCodeFmt = "SYSINIT_EXIT_CODE: %d"

// PrintExitCode prints the magic string communicating the exit code of the
// init to the writer (like [os.Stdout]).
func PrintExitCode(dst io.Writer, exitCode int) {
	// Ensure newlines before and after to avoid other writes messing up the
	// exit code communication as much as possible.
	msgFmt := "\n" + ExitCodeFmt + "\n"
	_, _ = fmt.Fprintf(dst, msgFmt, exitCode)
}

// PrintError prints an error to the writer (like [os.Stderr]).
func PrintError(dst io.Writer, err error) {
	_, _ = fmt.Fprintf(dst, "Error: %v\n", err)
}
