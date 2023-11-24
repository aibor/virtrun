package qemu

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"

	"github.com/aibor/virtrun/sysinit"
)

var panicRE = regexp.MustCompile(`^\[[0-9. ]+\] Kernel panic - not syncing: `)

var RCNotFoundErr = errors.New("no return code found in stdout")

// ParseStdout processes the input until the underlying writer is closed.
func ParseStdout(input io.Reader, output io.Writer, verbose bool) (int, error) {
	rc := 126
	// rcErr is unset once a return code is found.
	rcErr := RCNotFoundErr

	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		if panicRE.MatchString(line) {
			if rcErr != nil {
				rc = 125
			}
		} else if _, err := fmt.Sscanf(line, sysinit.RCFmt, &rc); err == nil {
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
