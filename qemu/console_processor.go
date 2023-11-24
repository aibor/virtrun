package qemu

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

// consoleProcessor is used to process input from a serial console and
// write it into a file.
type consoleProcessor struct {
	Path      string
	writePipe *os.File
	readPipe  io.ReadCloser
	output    io.WriteCloser
	ran       bool
}

// create creates the [os.Pipe] and returns the writing end. It also opens and
// truncates or creates the output file. Call [ConsoleProcessor.Close] in order
// to clean up the file descriptors after use.
func (p *consoleProcessor) create() (*os.File, error) {
	var err error
	p.readPipe, p.writePipe, err = os.Pipe()
	if err != nil {
		return nil, err
	}
	p.output, err = os.Create(p.Path)
	if err != nil {
		return nil, err
	}
	return p.writePipe, nil
}

// Close closes the file descriptors.
func (p *consoleProcessor) Close() error {
	var errs [3]error
	errs[0] = p.writePipe.Close()
	if !p.ran {
		errs[1] = p.readPipe.Close()
		errs[2] = p.output.Close()
	}
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
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
