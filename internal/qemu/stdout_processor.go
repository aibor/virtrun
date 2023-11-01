package qemu

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
)

// RCFmt is the format string for communicating the test results
//
// It is parsed in the qemu wrapper. Not present in the output if the test
// binary panicked.
const RCFmt = "INIT_RC: %d\n"

var panicRE = regexp.MustCompile(`^\[[0-9. ]+\] Kernel panic - not syncing: `)

// StdoutProcessor wraps [io.PipeWriter] and is used to determine the return
// code based on the standard output of the VM. It finds the well-known RC
// string for communication the return code from the guest. Call
// [StdoutProcessor.Close] in order to terminate the reader.
type StdoutProcessor struct {
	io.WriteCloser
	readPipe io.Reader
	output   io.Writer
	verbose  bool
	RC       int
	FoundRC  bool
}

// NewStdoutProcessor sets up a new StdoutProcessor.
func NewStdoutProcessor(output io.Writer, verbose bool) *StdoutProcessor {
	r, w := io.Pipe()
	return &StdoutProcessor{
		WriteCloser: w,
		readPipe:    r,
		output:      output,
		verbose:     verbose,
	}
}

// Run processes the input until the underlying writer is closed.
func (p *StdoutProcessor) Run() error {
	scanner := bufio.NewScanner(p.readPipe)
	for scanner.Scan() {
		line := scanner.Text()
		if panicRE.MatchString(line) {
			if !p.FoundRC {
				p.RC = 126
			}
		} else if _, err := fmt.Sscanf(line, RCFmt, &p.RC); err == nil {
			p.FoundRC = true
		}
		if !p.FoundRC || p.verbose {
			if _, err := fmt.Fprintln(p.output, line); err != nil {
				return err
			}
		}
	}
	return nil
}
