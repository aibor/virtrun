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
	"strconv"
	"strings"
	"time"

	"github.com/aibor/virtrun/internal/pipe"
)

const minAdditionalFileDescriptor = 3

// Console 0 is stderr. Console 1 is stdout.
const reservedPipes = 2

// AdditionalConsolePath returns guest's path to the additional console with the
// given index.
func AdditionalConsolePath(idx int) string {
	return pipe.Path(idx + reservedPipes)
}

// CommandSpec defines the parameters for a [Command].
type CommandSpec struct {
	// Path to the qemu-system binary
	Executable string

	// Path to the kernel to boot. The kernel should have Virtio-MMIO support
	// compiled in. If not, set the NoVirtioMMIO flag.
	Kernel string

	// Path to the initramfs to boot with. This is supposed to be a Initramfs
	// built with the initramfs sub package with an init that is built with
	// the sysinit sub package.
	Initramfs string

	// QEMU machine type to use. Depends on the QEMU binary used.
	Machine string

	// CPU type to use. Depends on machine type and QEMU binary used.
	CPU string

	// Number of CPUs for the guest.
	SMP uint64

	// Memory for the machine in MB.
	Memory uint64

	// Disable KVM support.
	NoKVM bool

	// Transport type for IO. This depends on machine type and the kernel.
	// TransportTypeIsa should always work, but will give only one slot for
	// microvm machine type. ARM type virt does not support ISA type at all.
	TransportType TransportType

	// ExtraArgs are  extra arguments that are passed to the QEMU command.
	// They must not interfere with the essential arguments set by the command
	// itself or an error will be returned on [Command.Run].
	ExtraArgs []Argument

	// Additional files attached to consoles besides the default one used for
	// stdout. They will be present in the guest system as "/dev/ttySx" or
	// "/dev/hvcx" where x is the index of the slice + 1. The guest is expected
	// to write all content base64 encoded.
	AdditionalConsoles []string

	// Arguments to pass to the init binary.
	InitArgs []string

	// Increase guest kernel logging.
	Verbose bool

	// ExitCodeParser parses the guest system's exit code from stdout.
	ExitCodeParser ExitCodeParser
}

// AddConsole adds an additional file to the QEMU command. This will be
// writable from the guest via the file descriptor number returned by this
// command. The guest is expected to write base64 encoded into the console.
func (c *CommandSpec) AddConsole(file string) string {
	c.AdditionalConsoles = append(c.AdditionalConsoles, file)
	return pipe.Path(c.numFDConsoles())
}

// Validate checks for known incompatibilities.
func (c *CommandSpec) Validate() error {
	if !c.TransportType.isKnown() {
		return &ArgumentError{
			"unknown transport type: " + c.TransportType.String(),
		}
	}

	switch c.Machine {
	case "microvm":
		switch {
		case c.TransportType == TransportTypePCI:
			return &ArgumentError{"microvm does not support pci transport"}
		case c.TransportType == TransportTypeISA &&
			len(c.AdditionalConsoles) > 0:
			return &ArgumentError{
				"microvm supports only one isa serial port, used for stdio",
			}
		}
	case "virt":
		if c.TransportType == TransportTypeISA {
			return &ArgumentError{"virt requires virtio-mmio"}
		}
	case "q35", "pc":
		if c.TransportType == TransportTypeMMIO {
			return &ArgumentError{
				c.Machine + " does not work with virtio-mmio",
			}
		}
	}

	return nil
}

// numFDConsoles returns the number of consoles attached via additional file
// descriptors. It is the number of additional consoles given by the user plus
// an additional one for a separate stdout channel.
func (c *CommandSpec) numFDConsoles() int {
	return len(c.AdditionalConsoles) + 1
}

// arguments compiles the argument list for the QEMU command.
func (c *CommandSpec) arguments() []Argument {
	args := []Argument{
		UniqueArg("kernel", c.Kernel),
		UniqueArg("initrd", c.Initramfs),
	}

	if c.Machine != "" {
		args = append(args, UniqueArg("machine", c.Machine))
	}

	if c.CPU != "" {
		args = append(args, UniqueArg("cpu", c.CPU))
	}

	if c.SMP != 0 {
		args = append(args, UniqueArg("smp", strconv.FormatUint(c.SMP, 10)))
	}

	if c.Memory != 0 {
		args = append(args, UniqueArg("m", strconv.FormatUint(c.Memory, 10)))
	}

	if !c.NoKVM {
		args = append(args, UniqueArg("enable-kvm", ""))
	}

	sharedDevices := map[TransportType]string{
		TransportTypePCI:  "virtio-serial-pci,max_ports=8",
		TransportTypeMMIO: "virtio-serial-device,max_ports=8",
	}
	if value, exists := sharedDevices[c.TransportType]; exists {
		args = append(args, RepeatableArg("device", value))
	}

	// Add stdout console.
	args = c.appendConsoleArgs(args, consoleArg{
		id:      "stdio",
		backend: "stdio",
	})

	// Write console output to file descriptors. Those are provided by the
	// [exec.Cmd.ExtraFiles].
	for idx := range c.numFDConsoles() {
		// FDs 0, 1, 2 are standard in, out, err, so start at 3.
		path := fdPath(minAdditionalFileDescriptor + idx)
		args = c.appendConsoleArgs(args, consoleArg{
			id:      fmt.Sprintf("con%d", idx),
			backend: "file",
			opts:    []string{"path=" + path},
		})
	}

	args = append(args,
		// Disable video output.
		UniqueArg("display", "none"),
		// Disable QEMU monitor.
		UniqueArg("monitor", "none"),
		// Guest must not reboot.
		UniqueArg("no-reboot"),
		// Disable all default devices.
		UniqueArg("nodefaults"),
		// Do not load any user config files.
		UniqueArg("no-user-config"),
	)

	args = append(args, c.ExtraArgs...)

	kernelCmdline := strings.Join(c.kernelCmdlineArgs(), " ")
	args = append(args, RepeatableArg("append", kernelCmdline))

	return args
}

// kernelCmdlineArgs reruns the kernel cmdline arguments.
func (c *CommandSpec) kernelCmdlineArgs() []string {
	cmdline := []string{
		"console=" + c.TransportType.ConsoleDeviceName(0),
		"panic=-1",
		"mitigations=off",
		"initcall_blacklist=ahci_pci_driver_init",
	}

	// ACPI is necessary for SMP. With a single CPU, we can disable it to speed
	// up the boot considerably.
	if c.SMP == 1 {
		cmdline = append(cmdline, "acpi=off")
	}

	if c.Verbose {
		cmdline = append(cmdline, "debug")
	} else {
		cmdline = append(cmdline, "quiet")
	}

	if len(c.InitArgs) > 0 {
		cmdline = append(cmdline, "--")
		cmdline = append(cmdline, c.InitArgs...)
	}

	return cmdline
}

type consoleArg struct {
	id      string
	backend string
	opts    []string
}

func (c *CommandSpec) appendConsoleArgs(
	args []Argument,
	console consoleArg,
) []Argument {
	var devArg Argument

	switch c.TransportType {
	case TransportTypeISA:
		devArg = RepeatableArg("serial", "chardev:"+console.id)
	case TransportTypePCI, TransportTypeMMIO:
		devArg = RepeatableArg("device", "virtconsole,chardev="+console.id)
	default: // Ignore invalid transport types.
		return args
	}

	chardevOpts := make([]string, 0, len(console.opts))
	chardevOpts = append(chardevOpts, console.backend, "id="+console.id)
	chardevOpts = append(chardevOpts, console.opts...)

	chardevArg := RepeatableArg("chardev", strings.Join(chardevOpts, ","))

	return append(args, chardevArg, devArg)
}

func fdPath(fd int) string {
	return fmt.Sprintf("/dev/fd/%d", fd)
}

// Command is single-use QEMU command.
type Command struct {
	name               string
	args               []string
	stdoutParser       stdoutParser
	additionalConsoles []string
}

// NewCommand builds the final [Command] with the given [CommandSpec].
func NewCommand(spec CommandSpec) (*Command, error) {
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
			ExitCodeParser: spec.ExitCodeParser,
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
