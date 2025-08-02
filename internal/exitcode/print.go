// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package exitcode

import (
	"fmt"
)

// Identifier is the identifier string for communicating an exit code via
// stdout.
const Identifier = "VIRTRUN_EXIT_CODE"

const format = Identifier + ": %d"

// Sprint creates the full exit code string with the given exit code.
func Sprint(exitCode int) string {
	return fmt.Sprintf(format, exitCode)
}
