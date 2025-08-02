// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"fmt"
	"io"

	"github.com/aibor/virtrun/internal/exitcode"
)

// PrintExitCode writes the exit code formatted into the given writer.
func PrintExitCode(writer io.Writer, exitCode int) {
	_, _ = fmt.Fprintln(writer, exitcode.Sprint(exitCode))
}
