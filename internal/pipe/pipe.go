// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package pipe

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"
)

const pathPrefix = "/dev/virtrun"

// Path creates the absolute host pipe path for the given port.
func Path(port int) string {
	return pathPrefix + strconv.Itoa(port)
}

// Pipe defines a data pipe that reads from any kind of input pipe (un/-named)
// [os.Pipe] or [io.Pipe] and writes it as is or modified into an output writer.
// It may not be used directly but passed to [Pipes.Run].
type Pipe struct {
	Name        string
	InputReader io.ReadCloser
	InputCloser io.Closer
	Output      io.Writer
	CopyFunc    CopyFunc
	MayBeSilent bool

	readOnce      sync.Once
	writeOnce     sync.Once
	readCloseErr  error
	writeCloseErr error

	done chan struct{}

	bytesWritten int64
	err          error
}

func (p *Pipe) String() string {
	return p.Name
}

// BytesWritten returns the bytes written by the pipe.
func (p *Pipe) BytesWritten() int64 {
	return p.bytesWritten
}

func (p *Pipe) close() error {
	writeErr := p.closeInput()

	p.readOnce.Do(func() {
		p.readCloseErr = p.InputReader.Close()
	})

	return errors.Join(writeErr, p.readCloseErr)
}

func (p *Pipe) closeInput() error {
	p.writeOnce.Do(func() {
		p.writeCloseErr = p.InputCloser.Close()
	})

	return p.writeCloseErr
}

func (p *Pipe) consume() {
	p.bytesWritten, p.err = p.CopyFunc(p.Output, p.InputReader)
}

func (p *Pipe) wait(deadline <-chan time.Time) error {
	_ = p.closeInput()
	select {
	case <-deadline:
		_ = p.close()
		<-p.done

		return ErrWaitTimeout
	case <-p.done:
	}

	if p.err != nil {
		return p.err
	}

	if p.bytesWritten == 0 && !p.MayBeSilent {
		return ErrNoOutput
	}

	return nil
}

// Pipes runs all given [Pipe] objects started with [Pipes.Run].
type Pipes struct {
	p []*Pipe
}

// Len returns the number of pipes.
func (p *Pipes) Len() int {
	return len(p.p)
}

// Run starts the given [Pipe].
//
// It runs until the writing end of the pipe is closed. It should.
func (p *Pipes) Run(pipe *Pipe) {
	if pipe.done != nil {
		panic("pipe already used")
	}

	pipe.done = make(chan struct{})

	go func() {
		defer close(pipe.done)
		defer pipe.close()

		pipe.consume()
	}()

	p.p = append(p.p, pipe)
}

// Wait waits for all consumers to terminate. It closes the readers forcefully
// if waiting exceeds the given timeout.
func (p *Pipes) Wait(timeout time.Duration) error {
	errs := make([]error, len(p.p))
	deadline := time.After(timeout)

	var finishers sync.WaitGroup

	for idx, pipe := range p.p {
		finishers.Go(func() {
			err := pipe.wait(deadline)
			if err != nil {
				errs[idx] = &Error{
					Name: pipe.Name,
					Err:  err,
				}
			}
		})
	}

	finishers.Wait()

	return errors.Join(errs...)
}

// BytesWritten returns the number of bytes written by name of the pipe.
func (p *Pipes) BytesWritten() map[string]int64 {
	bytesWritten := make(map[string]int64, len(p.p))
	for _, pipe := range p.p {
		bytesWritten[pipe.Name] = pipe.bytesWritten
	}

	return bytesWritten
}

// Close closes all readers.
//
// It interrupts active operations.
func (p *Pipes) Close() error {
	errs := []error{}

	for _, pipe := range p.p {
		err := pipe.close()
		if err != nil {
			errs = append(errs, fmt.Errorf("close %s: %w", pipe, err))
		}
	}

	return errors.Join(errs...)
}
