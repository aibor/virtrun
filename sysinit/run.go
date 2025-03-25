// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"fmt"
)

// ExitHandler is passed to [Run] and called with the first error a [Func]
// returns or nil if all [Func]s ran without error.
type ExitHandler func(err error)

// Func is a function run by [Run].
type Func func() error

// Run is the entry point for an actual init system.
//
// It runs the given functions and ensures proper shutdown of the system. It
// never returns. It must be run as PID 1, otherwise it panics immediately.
//
// The given [Func]s are run in the order given. They must not terminate the
// program (e.g. by [os.Exit]). Panics are recovered from before the
// [ExitHandler] runs.
//
// The given [ExitHandler] is run after the functions ran. It might be nil if
// they succeeded. It can be used for any post processing, like printing the
// exit code.
//
// A typical example that prints the exit code to stdout would be:
//
//	Run(
//		ExitCodeID.PrintExitCode,
//		[WithMountPoints]([SystemMountPoints]()),
//		[WithSymlinks]([DevSymlinks]()),
//		[WithInterfaceUp]("lo"),
//		[WithModules]("/lib/modules/*"),
//		[WithEnv]([EnvVars]{"PATH": "/data"}),
//		func() error {
//			// Optional additional custom setup functions.
//		},
//		func() error {
//			// Your actual main code.
//		},
//	)
func Run(exitHandler ExitHandler, funcs ...Func) {
	if !IsPidOne() {
		panic(ErrNotPidOne)
	}

	defer func() {
		if err := Poweroff(); err != nil {
			PrintError(err)
			panic(err)
		}
	}()

	err := runFuncs(funcs)

	if exitHandler != nil {
		exitHandler(err)
	}
}

func runFuncs(funcs []Func) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic during run: %v", r) //nolint:err113
		}
	}()

	for _, fn := range funcs {
		if err = fn(); err != nil {
			return err
		}
	}

	return nil
}
