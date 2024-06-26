// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package qemu

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"golang.org/x/sync/errgroup"
)

const minAdditionalFileDescriptor = 3

// Command defines the parameters for a single virtualized run.
type Command struct {
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
	SMP uint
	// Memory for the machine in MB.
	Memory uint
	// Disable KVM support.
	NoKVM bool
	// Transport type for IO. This depends on machine type and the kernel.
	// TransportTypeIsa should always work, but will give only one slot for
	// microvm machine type. ARM type virt does not support ISA type at all.
	TransportType TransportType
	// ExtraArgs are  extra arguments that are passed to the QEMU command.
	// They may not interfere with the essential arguments set by the command
	// itself or an error will be returned on [Command.Run].
	ExtraArgs []Argument
	// Additional files attached to consoles besides the default one used for
	// stdout. They will be present in the guest system as "/dev/ttySx" or
	// "/dev/hvcx" where x is the index of the slice + 1.
	AdditionalConsoles []string
	// Arguments to pass to the init binary.
	InitArgs []string
	// Print qemu command before running, increase guest kernel logging and
	// do not stop printing stdout when our RC string is found.
	Verbose bool
}

// NewCommand creates a new [Command] with defaults set to the given
// architecture. If it does not match the host architecture, the
// [Command.NoKVM] flag ist set.
// Supported architectures so far: amd64, arm64.
func NewCommand(arch string) (*Command, error) {
	cmd := Command{
		CPU:    "max",
		Memory: 256, //nolint:gomnd,mnd
		SMP:    1,
		NoKVM:  !KVMAvailableFor(arch),
		ExtraArgs: []Argument{
			UniqueArg("display", "none"),
			UniqueArg("monitor", "none"),
			UniqueArg("no-reboot", ""),
			UniqueArg("nodefaults", ""),
			UniqueArg("no-user-config", ""),
		},
	}

	switch arch {
	case "amd64":
		cmd.Executable = "qemu-system-x86_64"
		cmd.Machine = "q35"
		cmd.TransportType = TransportTypePCI
	case "arm64":
		cmd.Executable = "qemu-system-aarch64"
		cmd.Machine = "virt"
		cmd.TransportType = TransportTypeMMIO
	default:
		return nil, fmt.Errorf("arch not supported: %s", arch)
	}

	return &cmd, nil
}

// AddConsole adds an additional file to the QEMU command. This will be
// writable from the guest via the device name returned by this command.
// Console device number is starting at 1, as console 0 is the default stdout.
func (c *Command) AddConsole(file string) string {
	c.AdditionalConsoles = append(c.AdditionalConsoles, file)

	return c.TransportType.ConsoleDeviceName(uint8(len(c.AdditionalConsoles)))
}

// Validate checks for known incompatibilities.
func (c *Command) Validate() error {
	switch c.Machine {
	case "microvm":
		switch {
		case c.TransportType == TransportTypePCI:
			return errors.New("microvm does not support pci transport")
		case c.TransportType == TransportTypeISA && len(c.AdditionalConsoles) > 0:
			msg := "microvm supports only one isa serial port, used for stdio"

			return errors.New(msg)
		}
	case "virt":
		if c.TransportType == TransportTypeISA {
			return errors.New("virt requires virtio-mmio")
		}
	case "q35", "pc":
		if c.TransportType == TransportTypeMMIO {
			return fmt.Errorf("%s does not work with virtio-mmio", c.Machine)
		}
	}

	return nil
}

// ProcessGoTestFlags processes file related go test flags in
// [Command.InitArgs] and changes them, so the guest system's writes end up in
// the host systems file paths.
//
// It scans [Command.InitArgs] for coverage and profile related paths and
// replaces them with console path. The original paths are added as additional
// file descriptors to the [Command].
//
// It is required that the flags are prefixed with "test" and value is
// separated form the flag by "=". This is the format the "go test" tool
// invokes the test binary with.
func (c *Command) ProcessGoTestFlags() {
	// Only coverprofile has a relative path to the test pwd and can be
	// replaced immediately. All other profile files are relative to the actual
	// test running and need to be prefixed with -test.outputdir. So, collect
	// them and process them afterwards when "outputdir" is found.
	needsOutputDirPrefix := make([]int, 0)
	outputDir := ""

	for idx, posArg := range c.InitArgs {
		splits := strings.Split(posArg, "=")
		switch splits[0] {
		case "-test.coverprofile":
			splits[1] = "/dev/" + c.AddConsole(splits[1])
			c.InitArgs[idx] = strings.Join(splits, "=")
		case "-test.blockprofile",
			"-test.cpuprofile",
			"-test.memprofile",
			"-test.mutexprofile",
			"-test.trace":
			needsOutputDirPrefix = append(needsOutputDirPrefix, idx)

			continue
		case "-test.outputdir":
			outputDir = splits[1]

			fallthrough
		case "-test.gocoverdir":
			splits[1] = "/tmp"
			c.InitArgs[idx] = strings.Join(splits, "=")
		}
	}

	if outputDir != "" {
		for _, argsIdx := range needsOutputDirPrefix {
			splits := strings.Split(c.InitArgs[argsIdx], "=")
			path := filepath.Join(outputDir, splits[1])
			splits[1] = "/dev/" + c.AddConsole(path)
			c.InitArgs[argsIdx] = strings.Join(splits, "=")
		}
	}
}

// Args compiles the argument list for the QEMU command.
func (c *Command) Args() []Argument {
	a := []Argument{
		UniqueArg("kernel", c.Kernel),
		UniqueArg("initrd", c.Initramfs),
	}

	if c.Machine != "" {
		a = append(a, UniqueArg("machine", c.Machine))
	}

	if c.CPU != "" {
		a = append(a, UniqueArg("cpu", c.CPU))
	}

	if c.SMP != 0 {
		a = append(a, UniqueArg("smp", strconv.Itoa(int(c.SMP))))
	}

	if c.Memory != 0 {
		a = append(a, UniqueArg("m", strconv.Itoa(int(c.Memory))))
	}

	if !c.NoKVM {
		a = append(a, UniqueArg("enable-kvm", ""))
	}

	a = append(a, prepareConsoleArgs(c.TransportType)...)
	addConsoleArgs := consoleArgsFunc(c.TransportType)

	// Add stdout console.
	a = append(a, addConsoleArgs(1)...)

	// Write console output to file descriptors. Those are provided by the
	// [exec.Cmd.ExtraFiles].
	for idx := range c.AdditionalConsoles {
		// FDs 0, 1, 2 are standard in, out, err, so start at 3.
		a = append(a, addConsoleArgs(minAdditionalFileDescriptor+idx)...)
	}

	a = append(a, c.ExtraArgs...)

	kernelCmdline := strings.Join(c.kernelCmdlineArgs(), " ")
	a = append(a, RepeatableArg("append", kernelCmdline))

	return a
}

// kernelCmdlineArgs reruns the kernel cmdline arguments.
func (c *Command) kernelCmdlineArgs() []string {
	cmdline := []string{
		"console=" + c.TransportType.ConsoleDeviceName(0),
		"panic=-1",
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

func fdPath(fd int) string {
	return fmt.Sprintf("/dev/fd/%d", fd)
}

func prepareConsoleArgs(transportType TransportType) []Argument {
	switch transportType {
	case TransportTypePCI:
		return []Argument{
			RepeatableArg("device", "virtio-serial-pci,max_ports=8"),
		}
	case TransportTypeMMIO:
		return []Argument{
			RepeatableArg("device", "virtio-serial-device,max_ports=8"),
		}
	default: // Ignore invalid transport types.
		return nil
	}
}

func consoleArgsFunc(transportType TransportType) func(int) []Argument {
	switch transportType {
	case TransportTypeISA:
		return func(fd int) []Argument {
			return []Argument{
				RepeatableArg("serial", "file:"+fdPath(fd)),
			}
		}
	case TransportTypePCI, TransportTypeMMIO:
		return func(fd int) []Argument {
			vcon := fmt.Sprintf("vcon%d", fd)
			chardev := fmt.Sprintf("file,id=%s,path=%s", vcon, fdPath(fd))
			device := "virtconsole,chardev=" + vcon

			return []Argument{
				RepeatableArg("chardev", chardev),
				RepeatableArg("device", device),
			}
		}
	default: // Ignore invalid transport types.
		return func(_ int) (_ []Argument) { return }
	}
}

// Run the QEMU command with the given context.
//
// The final QEMU command is constructed, console processors are setup and the
// command is executed. A return code is returned. It can only be 0 if the
// guest system correctly communicated a 0 value via stdout. In any other case,
// a non 0 value is returned. If no error is returned, the value was received
// by the guest system.
func (c *Command) Run(ctx context.Context, stdout, stderr io.Writer) (int, error) {
	args, err := BuildArgumentStrings(c.Args())
	if err != nil {
		return 1, err
	}

	cmd := exec.CommandContext(ctx, c.Executable, args...)
	cmd.Stderr = stderr

	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		return 1, fmt.Errorf("get cmd stdout: %v", err)
	}

	// Create console output processors that fix line endings by stripping "\r".
	// Append the write end of the console processor pipe as extra file, so it
	// is present as additional file descriptor which can be used with the
	// "file" backend for QEMU console devices. [consoleProcessor.run] reads
	// from the read end of the pipe, cleans the output and writes it into
	// the actual target file on the host.
	processors, err := setupConsoleProcessors(c.AdditionalConsoles)
	if err != nil {
		return 1, err
	}
	defer processors.Close()

	var processorsGroup errgroup.Group

	for _, processor := range processors {
		cmd.ExtraFiles = append(cmd.ExtraFiles, processor.WritePipe)
		processorsGroup.Go(processor.run)
	}

	if err := cmd.Start(); err != nil {
		return 1, fmt.Errorf("start: %v", err)
	}

	// We need to process cmd stdout in any case to get our return code from
	// the command. If the caller did not pass any output writer, discard it.
	if stdout == nil {
		stdout = io.Discard
	}

	// Process output until the outPipe closes which happens automatically
	// at program termination. Error should be reported, but should not
	// terminate immediately. There might be more severe errors that following,
	// like process execution or persistent IO errors.
	rc, rcErr := ParseStdout(outPipe, stdout, c.Verbose)

	// Collect process information.
	if err := cmd.Wait(); err != nil {
		rc = 1

		return rc, fmt.Errorf("qemu: %v", err)
	}

	// Close console processors, so possible errors can be collected.
	processors.Close()

	if err := processorsGroup.Wait(); err != nil {
		return 1, fmt.Errorf("processor error: %v", err)
	}

	if rcErr != nil {
		return 1, rcErr
	}

	return rc, nil
}

// KVMAvailableFor checks if KVM support is available for the given
// architecture.
func KVMAvailableFor(arch string) bool {
	if runtime.GOARCH != arch {
		return false
	}

	f, err := os.OpenFile("/dev/kvm", os.O_WRONLY, 0)
	_ = f.Close()

	return err == nil
}
