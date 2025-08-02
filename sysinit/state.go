// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"fmt"
	"io"
	"slices"

	"github.com/aibor/virtrun/internal/exitcode"
)

// CleanupFunc defines a function that runs on cleanup after all [Func] ran.
type CleanupFunc func() error

// State keeps state of the current system.
type State struct {
	cleanupFns []CleanupFunc
	exitCode   *int
}

// Cleanup register a cleanup.
//
// Cleanup function run in the reverse order they are added. Errors are logged
// to the default logger.
func (s *State) Cleanup(fn CleanupFunc) {
	s.cleanupFns = append(s.cleanupFns, fn)
}

// SetExitCode sets the exit code to be written on successful run.
//
// If none is set or any errors occurred no exit code is written.
func (s *State) SetExitCode(exitCode int) {
	s.exitCode = &exitCode
}

func (s *State) doCleanup(errHandler func(error)) {
	slices.Reverse(s.cleanupFns)

	for _, fn := range s.cleanupFns {
		if err := fn(); err != nil {
			errHandler(fmt.Errorf("cleanup: %w", err))
		}
	}
}

func (s *State) printExitCode(writer io.Writer) {
	if s.exitCode != nil {
		_, _ = fmt.Fprintln(writer, exitcode.Sprint(*s.exitCode))
	}
}

func cleanupFunc(errHandler func(error)) Func {
	return func(state *State) error {
		state.doCleanup(errHandler)
		return nil
	}
}

func exitCodeFunc(writer io.Writer) Func {
	return func(state *State) error {
		state.printExitCode(writer)
		return nil
	}
}
