// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package qemu

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"golang.org/x/sync/errgroup"
)

const minAdditionalFileDescriptor = 3

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
	// "/dev/hvcx" where x is the index of the slice + 1.
	AdditionalConsoles []string

	// Arguments to pass to the init binary.
	InitArgs []string

	// Increase guest kernel logging.
	Verbose bool

	// ExitCodeParser parses the guest system's exit code from stdout.
	ExitCodeParser ExitCodeParser
}

// AddConsole adds an additional file to the QEMU command. This will be
// writable from the guest via the device name returned by this command.
// Console device number is starting at 1, as console 0 is the default stdout.
func (c *CommandSpec) AddConsole(file string) string {
	c.AdditionalConsoles = append(c.AdditionalConsoles, file)
	return c.TransportType.ConsoleDeviceName(uint(len(c.AdditionalConsoles)))
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
	args = c.appendConsoleArgs(args, console{
		id:      "stdio",
		backend: "stdio",
	})

	// Write console output to file descriptors. Those are provided by the
	// [exec.Cmd.ExtraFiles].
	for idx := range c.AdditionalConsoles {
		// FDs 0, 1, 2 are standard in, out, err, so start at 3.
		path := fdPath(minAdditionalFileDescriptor + idx)
		args = c.appendConsoleArgs(args, console{
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

type console struct {
	id      string
	backend string
	opts    []string
}

func (c *CommandSpec) appendConsoleArgs(
	args []Argument,
	console console,
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

	chardevOpts := []string{console.backend, "id=" + console.id}
	chardevOpts = append(chardevOpts, console.opts...)

	chardevArg := RepeatableArg("chardev", strings.Join(chardevOpts, ","))

	return append(args, chardevArg, devArg)
}

func fdPath(fd int) string {
	return fmt.Sprintf("/dev/fd/%d", fd)
}

// Command is single-use QEMU command.
type Command struct {
	name         string
	args         []string
	stdoutParser stdoutParser

	consoleOutput []string
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

	if spec.ExitCodeParser == nil {
		return nil, &ArgumentError{"ExitCodeScanFunc must not be empty"}
	}

	cmd := &Command{
		name:          spec.Executable,
		args:          cmdArgs,
		consoleOutput: spec.AdditionalConsoles,
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
	outputs := []io.Closer{}

	defer func() {
		for _, closer := range outputs {
			closeLog(closer)
		}
	}()

	var consoleProcessors errgroup.Group

	cmd := exec.CommandContext(ctx, c.name, c.args...)
	cmd.Stdin = stdin
	cmd.Stderr = stderr

	for _, path := range c.consoleOutput {
		dst, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("output file: %w", err)
		}

		outputs = append(outputs, dst)

		readPipe, writePipe, err := os.Pipe()
		if err != nil {
			return fmt.Errorf("pipe: %w", err)
		}

		// Append the write end of the console processor pipe as extra file, so
		// it is present as additional file descriptor which can be used with
		// the "file" backend for QEMU console devices. The processor reads from
		// the read end of the pipe, cleans the output and writes it into the
		// actual target file on the host.
		cmd.ExtraFiles = append(cmd.ExtraFiles, writePipe)

		consoleProcessor := consoleProcessor{
			dst: dst,
			src: readPipe,
		}

		consoleProcessors.Go(func() error {
			if err := consoleProcessor.run(); err != nil {
				return &ConsoleError{
					Name: path,
					Err:  err,
				}
			}

			return nil
		})
	}

	runErr := run(cmd, stdout, c.stdoutParser)
	consoleProcessorErr := consoleProcessors.Wait()

	if runErr != nil {
		return runErr
	}

	if consoleProcessorErr != nil {
		return fmt.Errorf("processors: %w", consoleProcessorErr)
	}

	return nil
}

func run(cmd *exec.Cmd, stdout io.Writer, stdoutParser stdoutParser) error {
	defer func() {
		for _, f := range cmd.ExtraFiles {
			closeLog(f)
		}
	}()

	// The default cancel function set by [exec.CommandContext] sends SIGKILL
	// to the process. This makes it impossible for QEMU to shutdown gracefully
	// which messes up terminal stdio and leaves the terminal in a broken state.
	cmd.Cancel = func() error {
		return cmd.Process.Signal(os.Interrupt)
	}

	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	stdoutProcessor := consoleProcessor{
		dst: stdout,
		src: outPipe,
		fn:  stdoutParser.Parse,
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start: %w", err)
	}

	processorErr := stdoutProcessor.run()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("command: %w", wrapExitError(err))
	}

	if processorErr != nil {
		return &ConsoleError{
			Name: "stdout",
			Err:  processorErr,
		}
	}

	return stdoutParser.GuestSuccessful()
}

func closeLog(closer io.Closer) {
	if err := closer.Close(); err != nil {
		slog.Error(
			"Failed to close qemu output listener",
			slog.Any("error", err),
		)
	}
}

func wrapExitError(err error) error {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return &CommandError{
			Err:      err,
			ExitCode: exitErr.ExitCode(),
		}
	}

	return err
}
