// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package qemu

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
)

// consoleProcessor is used to process input from a serial console and
// write it into a file.
type consoleProcessor struct {
	Path      string
	WritePipe *os.File
	readPipe  io.ReadCloser
	output    io.WriteCloser
	ran       bool
}

type consoleProcessors []*consoleProcessor

// Close closes all running processors.
func (p *consoleProcessors) Close() error {
	errs := make([]error, 0)

	for _, p := range *p {
		errs = append(errs, p.Close())
	}

	return errors.Join(errs...)
}

// newConsoleProcessor creates a new consoleProcessor and its required pipes.
// It also opens and truncates or creates the output file. Call
// [ConsoleProcessor.Close] in order to clean up the file descriptors after use.
func newConsoleProcessor(path string) (*consoleProcessor, error) {
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	output, err := os.Create(path)
	if err != nil {
		_ = writePipe.Close()
		_ = readPipe.Close()

		return nil, err
	}

	processor := &consoleProcessor{
		Path:      path,
		WritePipe: writePipe,
		readPipe:  readPipe,
		output:    output,
	}

	return processor, nil
}

// Close closes the file descriptors.
func (p *consoleProcessor) Close() error {
	var errs []error

	errs = append(errs, p.WritePipe.Close())
	if !p.ran {
		errs = append(errs, p.readPipe.Close())
		errs = append(errs, p.output.Close())
	}

	return errors.Join(errs...)
}

// run process the input. It blocks and returns once [io.EOF] is received,
// which happens when [ConsoleProcessor.Close] is called.
func (p *consoleProcessor) run() error {
	defer p.output.Close()
	defer p.readPipe.Close()
	p.ran = true

	scanner := bufio.NewScanner(p.readPipe)
	for scanner.Scan() {
		_, err := p.output.Write(append(scanner.Bytes(), byte('\n')))
		if err != nil {
			return fmt.Errorf("serial processor run %s: %v", p.Path, err)
		}
	}

	return nil
}

func setupConsoleProcessors(consolePaths []string) (consoleProcessors, error) {
	// Collect processors so they can be easily closed.
	processors := make([]*consoleProcessor, 0)

	for _, console := range consolePaths {
		processor, err := newConsoleProcessor(console)
		if err != nil {
			return nil, fmt.Errorf("create processor %s: %v", console, err)
		}

		processors = append(processors, processor)
	}

	return processors, nil
}
