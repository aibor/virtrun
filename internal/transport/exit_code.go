// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package transport

import (
	"fmt"
)

// Identifier is the identifier string for communicating an exit code via
// stdout.
const Identifier = "VIRTRUN_EXIT_CODE"

const format = Identifier + ": %d"

// FormatExitCode creates the full exit code string with the given exit code.
func FormatExitCode(exitCode int) string {
	return fmt.Sprintf(format, exitCode)
}

// ParseExitCode parses the given string for the exit code.
func ParseExitCode(line []byte) (int, bool) {
	var exitCode int

	_, err := fmt.Sscanf(string(line), format, &exitCode)

	return exitCode, err == nil
}
