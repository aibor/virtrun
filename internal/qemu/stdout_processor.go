package qemu

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
)

// RCFmt is the format string for communicating the test results
//
// It is parsed in the qemu wrapper. Not present in the output if the test
// binary panicked.
const RCFmt = "INIT_RC: %d\n"

var (
	// GuestNoRCFoundErr is returned if no return code matching the [RCFmt] is
	// found and no other error is found.
	GuestNoRCFoundErr = errors.New("guest did not print init return code")
	// GuestPanicErr is returned if a kernel panic occurred in the guest
	// system.
	GuestPanicErr = errors.New("guest system panicked")
	// GuestOomErr is returned if the guest system ran out of memory.
	GuestOomErr = errors.New("guest system ran out of memory")
)

var (
	panicRE = regexp.MustCompile(`^\[[0-9. ]+\] Kernel panic - not syncing: `)
	oomRE   = regexp.MustCompile(`^\[[0-9. ]+\] Out of memory: `)
)

// ParseStdout processes the input until the underlying writer is closed.
func ParseStdout(input io.Reader, output io.Writer, verbose bool) (int, error) {
	var rc int

	// rcErr is unset once a return code is found.
	rcErr := GuestNoRCFoundErr

	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case oomRE.MatchString(line):
			rcErr = GuestOomErr
		case panicRE.MatchString(line):
			rcErr = GuestPanicErr
		case rcErr == GuestNoRCFoundErr:
			if _, err := fmt.Sscanf(line, RCFmt, &rc); err == nil {
				rcErr = nil
			}
		}
		if rcErr != nil || verbose {
			if _, err := fmt.Fprintln(output, line); err != nil {
				return rc, err
			}
		}
	}

	return rc, rcErr
}
