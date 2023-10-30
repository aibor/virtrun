package internal

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

// RCParser wraps [io.PipeWriter] and is used to find our well-known RC
// string for communication the return code from the guest. Call
// [RCParser.Close] in order to terminate the reader.
type RCParser struct {
	io.WriteCloser
	scanner *bufio.Scanner
	output  io.Writer
	verbose bool
	RC      int
	FoundRC bool
}

// NewRCParser sets up a new RCParser.
func NewRCParser(output io.Writer, verbose bool) *RCParser {
	r, w := io.Pipe()
	return &RCParser{
		WriteCloser: w,
		scanner:     bufio.NewScanner(r),
		output:      output,
		verbose:     verbose,
	}
}

// Run processes the input until the underlying writer is closed.
func (p *RCParser) Run() error {
	for p.scanner.Scan() {
		line := p.scanner.Text()
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
