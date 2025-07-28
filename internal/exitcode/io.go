// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package exitcode

import (
	"fmt"
	"io"
	"strings"
)

// Identifier is the identifier string for communicating an exit code via
// stdout.
const Identifier = "VIRTRUN_EXIT_CODE"

// Fprint writes the full exit code line with the given exit code into the given
// writer.
func Fprint(w io.Writer, exitCode int) (int, error) {
	return fmt.Fprintf(w, format()+"\n", exitCode) //nolint:wrapcheck
}

// Parse parses the given string for the exit code.
//
// The identifier can be anywhere in the string. It does not need to be at the
// beginning. Returns the exit code and whether it was found in the given
// string.
func Parse(str string) (int, bool) {
	start := strings.Index(str, Identifier)
	if start < 0 {
		return 0, false
	}

	format := format()

	var exitCode int

	if _, err := fmt.Sscanf(str[start:], format, &exitCode); err != nil {
		return 0, false
	}

	return exitCode, true
}

func format() string {
	return string(Identifier) + ": %d"
}
