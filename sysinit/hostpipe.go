// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aibor/virtrun/internal/pipe"
)

// hostPipeStderr is the host pipe port reserved for plain text stderr and
// kernel default console.
const hostPipeStderr = 0

// hostPipeStdout is the host pipe port connected to the user's stdout.
const hostPipeStdout = 1

// WithHostPipes returns a setup [Func] that sets up encoded named pipes for
// communication with the host.
func WithHostPipes() Func {
	return func(state *State) error {
		// Setup host pipes for connected additional consoles that provide
		// encoded file content transmission to the host.
		hostPipes, err := OpenConsolePipes(state)
		if err != nil {
			return fmt.Errorf("create host pipes: %w", err)
		}

		state.Cleanup(func() error {
			// Let the host pipe writers finish. On busy hosts they might have a
			// write in flight even if the input is already done.
			if err := hostPipes.Wait(time.Second); err != nil {
				return fmt.Errorf("host pipes: %w", err)
			}

			return nil
		})

		return nil
	}
}

// WithStdoutHostPipe returns a setup [Func] that sets up an encoded pipe for
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
			_ = stdout.Close()

			return nil
		})

		return nil
	}
}

// OpenConsolePipes creates named pipes (fifos) in the file system and starts
// readers that write all data encoded into connected consoles.
//
// No pipe is created for the console used by stdout (hvc0/ttyS0), so only
// additional consoles are processed. If virto consoles are present (/dev/hvc*)
// then only those are used. Otherwise serial consoles (/dev/ttyS*) are used.
//
// The ID of the created host pipes matches the ID/port of the console it writes
// to, e.g. /dev/virtrun1 writes to /dev/hvc1.
func OpenConsolePipes(state *State) (*pipe.Pipes, error) {
	consoles, err := connectedConsoles()
	if err != nil {
		return nil, err
	}

	hostPipes := &pipe.Pipes{}

	for _, console := range consoles {
		var mayBeSilent bool

		switch console.port {
		// Skip hvc0/ttyS0 which is used for stderr/kernel log.
		case hostPipeStderr:
			continue
		// Allow no output for stdout.
		case hostPipeStdout:
			mayBeSilent = true
		}

		backend, err := os.OpenFile(console.path, os.O_WRONLY, 0)
		if err != nil {
			_ = hostPipes.Close()
			return nil, fmt.Errorf("console: %w", err)
		}

		path := pipe.Path(console.port)

		fifo, fifoStop, err := CreateNamedPipe(path)
		if err != nil {
			_ = backend.Close()
			_ = hostPipes.Close()

			return nil, fmt.Errorf("named pipe: %w", err)
		}

		hostPipe := &pipe.Pipe{
			Name:        path,
			InputReader: fifo,
			InputCloser: fifoStop,
			Output:      backend,
			CopyFunc:    pipe.Encode,
			MayBeSilent: mayBeSilent,
		}

		hostPipes.Run(hostPipe)

		state.Cleanup(backend.Close)
	}

	return hostPipes, nil
}

// CreateNamedPipe creates a named pipe (fifo(7)) at the given path and returns
// the reading and writing end of the pipe.
//
// Closing one of the ends results in an [io.EOF] unless there is another
// reader/writer that opened the pipe via the file system path.
//
// The caller is responsible for removing the fifo in the file system once it is
// not needed anymore.
func CreateNamedPipe(path string) (io.ReadCloser, io.WriteCloser, error) {
	if err := mkfifo(path); err != nil {
		return nil, nil, err
	}

	// Opening a single end of a pipe blocks until the other end is opened.
	// The R/W fd helps to open both pipe ends immediately. Another option
	// would be to open one of the ends in another go routine.
	temp, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("helper: %w", err)
	}
	defer temp.Close()

	reader, err := os.OpenFile(path, os.O_RDONLY|O_CLOEXEC, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("reader: %w", err)
	}

	writer, err := os.OpenFile(path, os.O_WRONLY|O_CLOEXEC, 0)
	if err != nil {
		_ = reader.Close()
		return nil, nil, fmt.Errorf("writer: %w", err)
	}

	return reader, writer, nil
}
