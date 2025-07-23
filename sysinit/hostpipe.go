// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aibor/virtrun/internal/pipe"
)

// WithHostPipes returns a setup [Func] that sets up encoded named pipes for
// communication with the host.
func WithHostPipes() Func {
	return func(state *State) error {
		// Setup host pipes for connected additional consoles that provide
		// encoded file content transmission to the host.
		hostPipes, err := OpenConsolePipes()
		if err != nil {
			return fmt.Errorf("create host pipes: %w", err)
		}

		state.Cleanup(func() error {
			// Let the host pipe writers finish. On busy hosts they might have a
			// write in flight even if the input is already done.
			ctx := context.Background()

			ctx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()

			if err := hostPipes.Wait(ctx); err != nil {
				return fmt.Errorf("host pipes: %w", err)
			}

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
// to, e.g. /dev/host_pipe1 writes to /dev/hvc1.
func OpenConsolePipes() (*pipe.Pipes, error) {
	consoles, err := connectedConsoles()
	if err != nil {
		return nil, err
	}

	hostPipes := &pipe.Pipes{}

	for _, console := range consoles {
		// Skip hvc0/ttyS0 which is used for stdout.
		if console.port == 0 {
			continue
		}

		backend, err := os.OpenFile(console.path, os.O_WRONLY, 0)
		if err != nil {
			_ = hostPipes.Close()
			return nil, fmt.Errorf("console: %w", err)
		}

		path := pipe.Path(console.port)

		fifo, fifoStop, err := newNamedPipe(path)
		if err != nil {
			_ = backend.Close()
			_ = hostPipes.Close()

			return nil, fmt.Errorf("named pipe: %w", err)
		}

		encoder := pipe.Encoder(backend)
		hostPipe := pipe.New(path, encoder, fifo, fifoStop)
		hostPipes.Run(hostPipe)
	}

	return hostPipes, nil
}

// newNamedPipe creates a named pipe (fifo(7)) at the given path and returns the
// reading and writing end of the pipe.
//
// Closing one of the ends results in an [io.EOF] unless there is another
// reader/writer that opened the pipe via the file system path.
//
// The caller is responsible for removing the fifo in the file system once it is
// not needed anymore.
func newNamedPipe(path string) (io.ReadCloser, io.WriteCloser, error) {
	if err := mkfifo(path); err != nil {
		return nil, nil, err
	}

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
