// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"fmt"
	"log"
)

// Func is a function run by [Run].
type Func func(*State) error

// Run is the entry point for an actual init system.
//
// It runs the given functions and ensures proper shutdown of the system. It
// never returns. It must be run as PID 1, otherwise it panics immediately.
//
// For proper communication with the virtrun host component /dev, /sys and /proc
// are mounted.
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
//
// Pay attention to the proper order: symlinks should be created after the
// dependent mounts.
func Run(exitHandler ExitHandler, funcs ...Func) {
	if !IsPidOne() {
		panic(ErrNotPidOne)
	}

	allFns := []Func{
		WithMountPoints(essentialMountPoints()),
	}
	allFns = append(allFns, funcs...)

	defer func() {
		if err := Poweroff(); err != nil {
			log.Print("ERROR poweroff: ", err.Error())
		}
	}()

	run(exitHandler, allFns)
}

func run(exitHandler ExitHandler, funcs []Func) {
	err := runFuncs(funcs)

	if exitHandler != nil {
		exitHandler(err)
	}
}

func runFuncs(funcs []Func) (err error) {
	defer func() {
		rec := recover()
		if rec == nil {
			return
		}

		if recoveredErr, ok := rec.(error); ok {
			err = fmt.Errorf("%w: %w", ErrPanic, recoveredErr)
		} else {
			err = fmt.Errorf("%w: %v", ErrPanic, rec)
		}
	}()

	state := new(State)

	for _, fn := range funcs {
		if err = fn(state); err != nil {
			return err
		}
	}

	state.doCleanup()

	return nil
}
