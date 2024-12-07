// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
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
	"slices"
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

	// ExitCodeFmt defines the format of the line communicating the exit code
	// from the guest. It must contain exactly one integer verb
	// (probably "%d").
	ExitCodeFmt string
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

	addConsoleArgs := consoleArgsFunc(c.TransportType)

	// Add stdout console.
	args = append(args, addConsoleArgs("stdio")...)

	// Write console output to file descriptors. Those are provided by the
	// [exec.Cmd.ExtraFiles].
	for idx := range c.AdditionalConsoles {
		// FDs 0, 1, 2 are standard in, out, err, so start at 3.
		path := fdPath(minAdditionalFileDescriptor + idx)
		args = append(args, addConsoleArgs("file", "path="+path)...)
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

	if !c.Verbose {
		cmdline = append(cmdline, "quiet")
	}

	if len(c.InitArgs) > 0 {
		cmdline = append(cmdline, "--")
		cmdline = append(cmdline, c.InitArgs...)
	}

	return cmdline
}

type Command struct {
	cmd          *exec.Cmd
	stdoutParser stdoutParser

	consoleOutput []string

	closer []io.Closer
}

// NewCommand builds the final [Command] with the given [CommandSpec].
func NewCommand(ctx context.Context, spec CommandSpec) (*Command, error) {
	// Do some simple input validation to catch most obvious issues.
	err := spec.Validate()
	if err != nil {
		return nil, err
	}

	cmdArgs, err := BuildArgumentStrings(spec.arguments())
	if err != nil {
		return nil, err
	}

	if spec.ExitCodeFmt == "" {
		return nil, &ArgumentError{"ExitCodeFmt must not be empty"}
	}

	cmd := &Command{
		cmd:           exec.CommandContext(ctx, spec.Executable, cmdArgs...),
		consoleOutput: spec.AdditionalConsoles,
		stdoutParser: stdoutParser{
			ExitCodeFmt: spec.ExitCodeFmt,
			Verbose:     spec.Verbose,
		},
	}

	// The default cancel function set by [exec.CommandContext] sends SIGKILL
	// to the process. This makes it impossible for QEMU to shutdown gracefully
	// which messes up terminal stdio and leaves the terminal in a broken state.
	cmd.cmd.Cancel = func() error {
		return cmd.cmd.Process.Signal(os.Interrupt)
	}

	return cmd, nil
}

// String prints the human readable string representation of the command.
//
// It just wraps [exec.Command.String].
func (c *Command) String() string {
	return c.cmd.String()
}

// stdoutProcessor creates a new [consoleProcessor] with the command's
// [stdoutParser].
func (c *Command) stdoutProcessor(dst io.Writer) (*consoleProcessor, error) {
	outPipe, err := c.cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	processor := &consoleProcessor{
		dst: dst,
		src: outPipe,
		fn:  c.stdoutParser.Parse,
	}

	return processor, nil
}

// consoleProcessor opens the console output file at path and returns the
// processor function that cleans the output from carriage returns.
func (c *Command) consoleProcessor(path string) (*consoleProcessor, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("open console output: %w", err)
	}

	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("pipe: %w", err)
	}

	processor := &consoleProcessor{
		dst: f,
		src: readPipe,
	}

	// Append the write end of the console processor pipe as extra file, so it
	// is present as additional file descriptor which can be used with the
	// "file" backend for QEMU console devices. The processor reads from the
	// read end of the pipe, cleans the output and writes it into the actual
	// target file on the host.
	c.cmd.ExtraFiles = append(c.cmd.ExtraFiles, writePipe)
	c.closer = append(c.closer, f, writePipe)

	return processor, nil
}

func (c *Command) close() {
	for _, closer := range slices.Backward(c.closer) {
		_ = closer.Close()
	}
}

// Run the [Command].
//
// Output processors are setup and the command is executed. Returns without
// error only if the guest system correctly communicated exit code 0. In any
// other case, an error is returned. If the QEMU command itself failed,
// a [CommandError] with the guest flag unset is returned. If the guest
// returned an error or failed a [CommandError] with guest flag set is
// returned.
func (c *Command) Run(stdout, stderr io.Writer) error {
	defer c.close()

	var processors errgroup.Group

	for _, path := range c.consoleOutput {
		processor, err := c.consoleProcessor(path)
		if err != nil {
			return err
		}

		processors.Go(processor.run)
	}

	c.cmd.Stderr = stderr

	stdoutProcessor, err := c.stdoutProcessor(stdout)
	if err != nil {
		return err
	}

	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("start: %w", err)
	}

	if err := stdoutProcessor.run(); err != nil {
		return fmt.Errorf("stdout parser: %w", err)
	}

	if err := c.cmd.Wait(); err != nil {
		return wrapExitError(err)
	}

	// Close all FDs so processors stop.
	for _, f := range c.cmd.ExtraFiles {
		_ = f.Close()
	}

	err = processors.Wait()
	if err != nil {
		return fmt.Errorf("processor wait: %w", err)
	}

	return c.stdoutParser.GuestSuccessful()
}

func wrapExitError(err error) error {
	var exitErr *exec.ExitError

	if !errors.As(err, &exitErr) {
		return fmt.Errorf("qemu command: %w", err)
	}

	return &CommandError{
		Err:      err,
		ExitCode: exitErr.ExitCode(),
	}
}
