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

var panicRE = regexp.MustCompile(`^\[[0-9. ]+\] Kernel panic - not syncing: `)

var RCNotFoundErr = errors.New("no return code found in stdout")

// ParseStdout processes the input until the underlying writer is closed.
func ParseStdout(input io.Reader, output io.Writer, verbose bool) (int, error) {
	var rc int
	// rcErr is unset once a return code is found.
	rcErr := RCNotFoundErr

	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		if panicRE.MatchString(line) {
			if rcErr != nil {
				rc = 126
			}
		} else if _, err := fmt.Sscanf(line, RCFmt, &rc); err == nil {
			rcErr = nil
		}
		if rcErr != nil || verbose {
			if _, err := fmt.Fprintln(output, line); err != nil {
				return rc, err
			}
		}
	}

	return rc, rcErr
}
