// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package qemu

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/aibor/virtrun/internal/pipe"
)

// Console 0 is stderr. Console 1 is stdout.
const reservedPipes = 2

// AdditionalConsolePath returns guest's path to the additional console with the
// given index.
func AdditionalConsolePath(idx int) string {
	return pipe.Path(idx + reservedPipes)
}

// Command is single-use QEMU command.
type Command struct {
	name               string
	args               []string
	stdoutParser       stdoutParser
	additionalConsoles []string
}

// NewCommand builds the final [Command] with the given [CommandSpec].
func NewCommand(spec CommandSpec, exitParser ExitCodeParser) (*Command, error) {
	// Do some simple input validation to catch most obvious issues.
	err := spec.Validate()
	if err != nil {
		return nil, err
	}

	cmdArgs, err := BuildArgumentStrings(spec.arguments())
	if err != nil {
		return nil, err
	}

	cmd := &Command{
		name:               spec.Executable,
		args:               cmdArgs,
		additionalConsoles: spec.AdditionalConsoles,
		stdoutParser: stdoutParser{
			ExitCodeParser: exitParser,
			Verbose:        spec.Verbose,
		},
	}

	return cmd, nil
}

// String prints the human readable string representation of the command.
func (c *Command) String() string {
	elems := append([]string{c.name}, c.args...)
	return strings.Join(elems, " ")
}

// Run the [Command] with the given [context.Context].
//
// Output processors are setup and the command is executed. Returns without
// error only if the guest system correctly communicated exit code 0. In any
// other case, an error is returned. If the QEMU command itself failed,
// a [CommandError] with the guest flag unset is returned. If the guest
// returned an error or failed a [CommandError] with guest flag set is
// returned.
func (c *Command) Run(
	ctx context.Context,
	stdin *os.File,
	stdout *os.File,
	stderr *os.File,
) error {
	outputFiles, err := openFiles(c.additionalConsoles)
	if err != nil {
		return err
	}
	defer cleanup(outputFiles)

	cmd := exec.CommandContext(ctx, c.name, c.args...)

	// The default cancel function set by [exec.CommandContext] sends SIGKILL
	// to the process. This makes it impossible for QEMU to shutdown gracefully
	// which messes up terminal stdio and leaves the terminal in a broken state.
	cmd.Cancel = func() error {
		return cmd.Process.Signal(os.Interrupt)
	}

	// The guest is supposed to use the first virtrun pipe as stdout for its
	// payload.
	cmd.ExtraFiles = append(cmd.ExtraFiles, stdout)

	// Additional console output.
	cmd.ExtraFiles = append(cmd.ExtraFiles, outputFiles...)

	cmd.Stdin = stdin
	cmd.Stderr = stderr

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("start: %w", err)
	}

	// The guest is supposed to only write errors and host communication like
	// the exit code into the default console. Thus, write the command's stdout
	// into stderr.
	_, err = c.stdoutParser.Copy(stderr, stdoutPipe)
	if err != nil {
		return err
	}

	err = cmd.Wait()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return &CommandError{
				Err:      err,
				ExitCode: exitErr.ExitCode(),
			}
		}

		return fmt.Errorf("command: %w", err)
	}

	guestExitCode, err := c.stdoutParser.Result()
	if err != nil {
		return &CommandError{
			Err:      err,
			Guest:    true,
			ExitCode: guestExitCode,
		}
	}

	return nil
}

func openFiles(paths []string) ([]*os.File, error) {
	outputs := []*os.File{}

	for _, path := range paths {
		output, err := os.Create(path)
		if err != nil {
			for _, c := range outputs {
				_ = c.Close()
			}

			return nil, err
		}

		outputs = append(outputs, output)
	}

	return outputs, nil
}

func cleanup[T io.Closer](closer []T) {
	for _, c := range closer {
		_ = c.Close()
	}
}
