// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package exitcode

import (
	"bytes"
	"fmt"
)

// Parse parses the given string for the exit code.
func Parse(b []byte) (int, bool) {
	var buf bytes.Reader

	buf.Reset(b)

	var exitCode int
	if _, err := fmt.Fscanf(&buf, format, &exitCode); err != nil {
		return 0, false
	}

	return exitCode, true
}
