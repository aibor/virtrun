// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package sysinit

import (
	"errors"
	"os"
	"testing"
)

// RunTests sets up the system, runs the tests and shuts down the system.
//
// Call it from your `TestMain` function. It wraps [testing.M.Run] and returns
// only in case of failure. It is an error if the process does not run with
// PID 1, since the intention of this library is to run test binaries in an
// isolated system.
func RunTests(m *testing.M, cfg Config) {
	err := Run(cfg, func() (int, error) {
		return m.Run(), nil
	})

	exitCode := 126
	if errors.Is(err, ErrNotPidOne) {
		exitCode = 127
	}

	PrintError(os.Stderr, err)

	os.Exit(exitCode)
}
