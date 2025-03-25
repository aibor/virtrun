// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"fmt"
	"io"
	"os"
)

// PrintError prints the error to [os.Stderr].
func PrintError(err error) {
	_, _ = FprintError(os.Stderr, err)
}

// FprintError prints the error to the given [io.Writer].
func FprintError(w io.Writer, e error) (int, error) {
	//nolint:wrapcheck
	return fmt.Fprintf(w, "Error: %v\n", e)
}

// PrintWarning prints the error as warning to [os.Stderr].
func PrintWarning(err error) {
	_, _ = FprintWarning(os.Stderr, err)
}

// FprintWarning prints the error as warning to the given [io.Writer].
func FprintWarning(w io.Writer, e error) (int, error) {
	//nolint:wrapcheck
	return fmt.Fprintf(w, "Warning: %v\n", e)
}
