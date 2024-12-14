// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"fmt"
	"os"
)

// ExitCodeFmt is the format string for communicating the test results
//
// The same format string must be configured for the [qemu.Command] so it is
// matched correctly.
const ExitCodeFmt = "SYSINIT_EXIT_CODE: %d"

// PrintExitCode prints the magic string communicating the exit code of the
// init to stdout.
func PrintExitCode(exitCode int) {
	// Ensure newlines before and after to avoid other writes messing up the
	// exit code communication as much as possible.
	msgFmt := "\n" + ExitCodeFmt + "\n"
	_, _ = fmt.Fprintf(os.Stdout, msgFmt, exitCode)
}

// PrintError prints the given error to stderr.
func PrintError(err error) {
	_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
}
