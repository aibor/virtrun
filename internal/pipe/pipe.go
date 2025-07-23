// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package pipe

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
)

// Pipe listens on a fifo(7) (named pipe) and reads all data written to
// it into the backend writer.
type Pipe struct {
	Name string

	src           io.Reader
	dst           Consumer
	done          chan struct{}
	closeRW       func() error
	stopListening func() error

	bytesRead int64
	err       error
}

func New(name string, dst Consumer, src io.ReadCloser, stop io.Closer) *Pipe {
	return &Pipe{
		Name: name,
		src:  src,
		dst:  dst,
		done: make(chan struct{}),
		closeRW: sync.OnceValue(func() error {
			return errors.Join(src.Close(), dst.Close())
		}),
		stopListening: sync.OnceValue(stop.Close),
	}
}

func (c *Pipe) String() string {
	return c.Name
}

func (c *Pipe) Close() error {
	return errors.Join(c.stopListening(), c.closeRW())
}

// run reads from the pipe and writes into the given writer.
//
// It blocks until done. It closes the writer and the pipe once done. Results
// are returned by calling [hostPipe.Wait]. Call [hostPipe.Close] to cancel the
// operation.
func (c *Pipe) run() {
	defer close(c.done)
	defer c.dst.Close()
	defer c.Close()

	c.bytesRead, c.err = c.dst.ReadFrom(c.src)
}

// Pipes provides data channel to send binary data to the host. It provides
// named pipes (fifos) for consumption by any process. They can be used only
// once and each pipe is closed when the writing process closes it.
type Pipes struct {
	fifoReaders []*Pipe
}

// Wait waits for all consumers to terminate. It closes the readers forcefully
// if waiting exceeds the given timeout.
func (c *Pipes) Wait(ctx context.Context) error {
	errCh := make(chan error)

	var cancelGroup sync.WaitGroup

	cancelGroup.Add(len(c.fifoReaders))

	for _, hostPipe := range c.fifoReaders {
		_ = hostPipe.stopListening()

		go func() {
			defer cancelGroup.Done()

			var err error

			select {
			case <-hostPipe.done:
				err = hostPipe.err
				if err == nil && hostPipe.bytesRead == 0 {
					err = ErrNoOutput
				}
			case <-ctx.Done():
				_ = hostPipe.Close()
				<-hostPipe.done

				err = ErrTimeout
			}

			if err != nil {
				errCh <- &Error{
					Name: hostPipe.Name,
					Err:  ErrNoOutput,
				}
			}
		}()
	}

	go func() {
		cancelGroup.Wait()
		close(errCh)
	}()

	errs := []error{}
	for err := range errCh {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

// Close closes all readers.
//
// It interrupts active operations.
func (c *Pipes) Close() error {
	errs := []error{}

	for _, hostPipe := range c.fifoReaders {
		_ = os.RemoveAll(hostPipe.Name)

		if err := hostPipe.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close %s: %w", hostPipe, err))
		}
	}

	return errors.Join(errs...)
}

func (c *Pipes) Run(hostPipe *Pipe) {
	go hostPipe.run()
	c.fifoReaders = append(c.fifoReaders, hostPipe)
}
