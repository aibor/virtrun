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

// ConsoleProcessor is used to process input from a serial console and
// write it into a file.
type ConsoleProcessor struct {
	WritePipe *os.File
	readPipe  io.ReadCloser
	output    io.WriteCloser
	ran       bool
}

type ConsoleProcessors []*ConsoleProcessor

// Close closes all running processors.
func (p *ConsoleProcessors) Close() error {
	errs := make([]error, 0)

	for _, p := range *p {
		errs = append(errs, p.Close())
	}

	return errors.Join(errs...)
}

// NewConsoleProcessor creates a new consoleProcessor and its required pipes.
// It also opens and truncates or creates the output file. Call
// [ConsoleProcessor.Close] in order to clean up the file descriptors after use.
func NewConsoleProcessor(output io.WriteCloser) (*ConsoleProcessor, error) {
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("pipe: %w", err)
	}

	processor := &ConsoleProcessor{
		WritePipe: writePipe,
		readPipe:  readPipe,
		output:    output,
	}

	return processor, nil
}

// Close closes the file descriptors.
func (p *ConsoleProcessor) Close() error {
	var errs []error

	errs = append(errs, p.WritePipe.Close())
	if !p.ran {
		errs = append(errs, p.readPipe.Close())
		errs = append(errs, p.output.Close())
	}

	return errors.Join(errs...)
}

// Run process the input. It blocks and returns once [io.EOF] is received,
// which happens when [ConsoleProcessor.Close] is called.
func (p *ConsoleProcessor) Run() error {
	defer p.readPipe.Close()
	p.ran = true

	scanner := bufio.NewScanner(p.readPipe)
	for scanner.Scan() {
		_, err := p.output.Write(append(scanner.Bytes(), byte('\n')))
		if err != nil {
			return fmt.Errorf("serial processor run: %w", err)
		}
	}

	return nil
}

func SetupConsoleProcessors(consoles []io.WriteCloser) (ConsoleProcessors, error) {
	// Collect processors so they can be easily closed.
	processors := make([]*ConsoleProcessor, 0)

	for _, console := range consoles {
		processor, err := NewConsoleProcessor(console)
		if err != nil {
			return nil, fmt.Errorf("create processor: %w", err)
		}

		processors = append(processors, processor)
	}

	return processors, nil
}
