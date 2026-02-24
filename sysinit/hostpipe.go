// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"fmt"
	"os"

	"github.com/aibor/virtrun/internal/pipe"
)

// hostPipeStdout is the host pipe port connected to the user's stdout.
const hostPipeStdout = 1

// WithHostPipes returns a setup [Func] that sets up consoles for communication
// with the host.
func WithHostPipes() Func {
	return func(state *State) error {
		err := SetupHostPipes(state)
		if err != nil {
			return fmt.Errorf("create host pipes: %w", err)
		}

		return nil
	}
}

// WithStdoutHostPipe returns a setup [Func] that sets up a separate stream for
// stdout. It replaces the original [os.Stdout].
func WithStdoutHostPipe() Func {
	return func(state *State) error {
		stdout, err := os.OpenFile(pipe.Path(hostPipeStdout), os.O_WRONLY, 0)
		if err != nil {
			return err
		}

		var oldStdout *os.File

		oldStdout, os.Stdout = os.Stdout, stdout

		state.Cleanup(func() error {
			os.Stdout = oldStdout
			return stdout.Close()
		})

		return nil
	}
}

// SetupHostPipes configures all present serial/virtual consoles and creates
// /dev/virtun* symlinks.
//
// If virto consoles are present (/dev/hvc*) then only those are used. Otherwise
// serial consoles (/dev/ttyS*) are used.
//
// The ID of the created host pipes matches the ID/port of the console it writes
// to, e.g. /dev/virtrun1 writes to /dev/hvc1.
func SetupHostPipes(state *State) error {
	consoles, err := connectedConsoles()
	if err != nil {
		return err
	}

	for _, console := range consoles {
		handle, err := fopen(console.path, O_WRONLY|O_NOCTTY|O_NDELAY, 0)
		if err != nil {
			return err
		}

		err = configureConsole(handle)
		if err != nil {
			_ = fclose(handle)
			return fmt.Errorf("configure %s: %w", console.path, err)
		}

		err = os.Symlink(console.path, pipe.Path(console.port))
		if err != nil {
			_ = fclose(handle)
			return fmt.Errorf("symlink pipe: %w", err)
		}

		state.Cleanup(func() error {
			return fclose(handle)
		})
	}

	return nil
}
