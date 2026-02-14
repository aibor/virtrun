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
	"time"

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
	stdin io.Reader,
	stdout, stderr io.Writer,
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

	pipes := guestPipes{}
	defer pipes.Close()

	// The guest is supposed to only write errors and host communication like
	// the exit code into the default console. Thus, write the command's stdout
	// into stderr.
	stderrPipe, err := pipes.addPipe(stderr, c.stdoutParser.Copy, true)
	if err != nil {
		return err
	}

	cmd.Stdin = stdin
	cmd.Stdout = stderrPipe
	cmd.Stderr = stderr

	// Append the write end of the console processor pipe as extra file, so
	// it is present as additional file descriptor which can be used with
	// the "file" backend for QEMU console devices. The processor reads from
	// the read end of the pipe, decodes the output and writes it into the
	// actual target writer

	// The guest is supposed to use the first virtrun pipe as stdout for its
	// payload.
	stdoutPipe, err := pipes.addPipe(stdout, pipe.DecodeLineBuffered, true)
	if err != nil {
		return err
	}

	cmd.ExtraFiles = append(cmd.ExtraFiles, stdoutPipe)

	// Additional console output.
	for _, output := range outputFiles {
		writer, err := pipes.addPipe(output, pipe.Decode, false)
		if err != nil {
			return err
		}

		cmd.ExtraFiles = append(cmd.ExtraFiles, writer)
	}

	runErr := cmd.Run()

	pipesErr := pipes.Wait(time.Second)

	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			return &CommandError{
				Err:      runErr,
				ExitCode: exitErr.ExitCode(),
			}
		}

		return fmt.Errorf("command: %w", runErr)
	}

	guestExitCode, err := c.stdoutParser.Result()
	if err != nil {
		return &CommandError{
			Err:      err,
			Guest:    true,
			ExitCode: guestExitCode,
		}
	}

	return pipesErr //nolint:wrapcheck
}

type guestPipes struct {
	pipe.Pipes
}

func (p *guestPipes) addPipe(
	output io.Writer,
	copyFn pipe.CopyFunc,
	maybeSilent bool,
) (*os.File, error) {
	pipeReader, pipeWriter, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("pipe: %w", err)
	}

	p.Run(&pipe.Pipe{
		Name:        pipe.Path(p.Len()),
		InputReader: pipeReader,
		InputCloser: pipeWriter,
		Output:      output,
		CopyFunc:    copyFn,
		MayBeSilent: maybeSilent,
	})

	return pipeWriter, nil
}

func openFiles(paths []string) ([]io.WriteCloser, error) {
	outputs := []io.WriteCloser{}

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
