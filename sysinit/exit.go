// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"io"
	"log"

	"github.com/aibor/virtrun/internal/exitcode"
)

// ExitHandler is passed to [Run] and called with the first error a [Func]
// returns or nil if all [Func]s ran without error.
type ExitHandler func(err error)

// ExitCodePrinter returns an [ExitHandler] that writes the exit code based on
// the given [ExitError] into the given writer.
func ExitCodePrinter(writer io.Writer) ExitHandler {
	return func(err error) {
		exitCode, isExitErr := exitcode.From(err)
		if err != nil && !isExitErr {
			log.Print("ERROR ", err.Error())
		}

		_, _ = exitcode.Fprint(writer, exitCode)
	}
}
