// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package qemu

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
)

var (
	panicRE = regexp.MustCompile(`^\[[0-9. ]+\] Kernel panic - not syncing: `)
	oomRE   = regexp.MustCompile(`^\[[0-9. ]+\] Out of memory: `)
)

type outputProcessor func() error

// parseStdout returns an [outputProcessor] that parses stdout from the guest.
//
// It detects kernel panics, OOM messages and most importantly it detects the
// exit code communicated by the guest via stdout. The processor stops when
// the src is closed. It returns a [CommandError] with Guest flag set if either
// an error is detected or the guest communicated a non zero exit code.
func parseStdout(
	dst io.Writer,
	src io.Reader,
	exitCodeFmt string,
	verbose bool,
) outputProcessor {
	return func() error {
		var exitCode int

		// guestErr is unset once an exit code is found in the output stream.
		guestErr := ErrGuestNoExitCodeFound

		scanner := bufio.NewScanner(src)
		for scanner.Scan() {
			line := scanner.Text()

			// Parse the output. Keep going after a match has been found, so
			// the following lines are printed as well and enhance the context
			// information in case of kernel error messages.
			switch {
			case oomRE.MatchString(line):
				guestErr = ErrGuestOom
			case panicRE.MatchString(line):
				guestErr = ErrGuestPanic
			case guestErr == ErrGuestNoExitCodeFound: //nolint:errorlint,err113
				_, err := fmt.Sscanf(line, exitCodeFmt, &exitCode)
				if err != nil {
					break
				}

				guestErr = nil
				if exitCode != 0 {
					guestErr = ErrGuestNonZeroExitCode
				}

				// Skip line printing once the init exit code has been found
				// unless the verbose flag is set.
				if !verbose {
					dst = nil
				}
			}

			err := writeLn(dst, scanner.Bytes())
			if err != nil {
				return err
			}
		}

		if scanner.Err() != nil {
			return scanner.Err()
		}

		return wrapGuestError(guestErr, exitCode)
	}
}

func wrapGuestError(err error, exitCode int) error {
	if err == nil {
		return nil
	}

	return &CommandError{
		Guest:    true,
		ExitCode: exitCode,
		Err:      err,
	}
}

// scrubCR creates a simple [outputProcessor] that just sanitizes line breaks.
//
// It returns the processor and a write pipe as *os.File. The caller is
// responsible to close the writePipe. This terminates the processor.
func scrubCR(dst io.Writer) (outputProcessor, *os.File, error) {
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		return nil, nil, fmt.Errorf("pipe: %w", err)
	}

	processor := func() error {
		defer readPipe.Close()

		// Carriage returns are removed by [bufio.ScanLines].
		scanner := bufio.NewScanner(readPipe)
		for scanner.Scan() {
			err := writeLn(dst, scanner.Bytes())
			if err != nil {
				return err
			}
		}

		return scanner.Err()
	}

	return processor, writePipe, nil
}

func writeLn(dst io.Writer, data []byte) error {
	// If the caller did not pass any output writer, discard it.
	if dst == nil {
		return nil
	}

	_, err := dst.Write(data)
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}

	_, err = dst.Write([]byte("\n"))
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}
