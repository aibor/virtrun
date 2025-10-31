// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package exitcode

import (
	"bytes"
	"fmt"
)

// Parse parses the given string for the exit code.
func Parse(line []byte) (int, bool) {
	var (
		buf      bytes.Reader
		exitCode int
	)

	buf.Reset(line)

	_, err := fmt.Fscanf(&buf, format, &exitCode)
	if err != nil {
		return 0, false
	}

	return exitCode, true
}
