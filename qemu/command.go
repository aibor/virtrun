package qemu

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/sync/errgroup"
)

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
	ExtraArgs Arguments
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
		Memory: 256,
		SMP:    1,
		NoKVM:  !KVMAvailableFor(arch),
		ExtraArgs: Arguments{
			ArgDisplay("none"),
			ArgMonitor("none"),
			UniqueArg("no-reboot"),
			UniqueArg("nodefaults"),
			UniqueArg("no-user-config"),
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
			return fmt.Errorf("microvm does not support pci transport.")
		case c.TransportType == TransportTypeISA && len(c.AdditionalConsoles) > 0:
			msg := "microvm supports only one isa serial port, used for stdio."
			return fmt.Errorf(msg)
		}
	case "virt":
		if c.TransportType == TransportTypeISA {
			return fmt.Errorf("virt requires virtio-mmio")
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
func (cmd *Command) ProcessGoTestFlags() {
	// Only coverprofile has a relative path to the test pwd and can be
	// replaced immediately. All other profile files are relative to the actual
	// test running and need to be prefixed with -test.outputdir. So, collect
	// them and process them afterwards when "outputdir" is found.
	needsOutputDirPrefix := make([]int, 0)
	outputDir := ""

	for idx, posArg := range cmd.InitArgs {
		splits := strings.Split(posArg, "=")
		switch splits[0] {
		case "-test.coverprofile":
			splits[1] = "/dev/" + cmd.AddConsole(splits[1])
			cmd.InitArgs[idx] = strings.Join(splits, "=")
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
			cmd.InitArgs[idx] = strings.Join(splits, "=")
		}
	}

	if outputDir != "" {
		for _, argsIdx := range needsOutputDirPrefix {
			splits := strings.Split(cmd.InitArgs[argsIdx], "=")
			path := filepath.Join(outputDir, splits[1])
			splits[1] = "/dev/" + cmd.AddConsole(path)
			cmd.InitArgs[argsIdx] = strings.Join(splits, "=")
		}
	}
}

// Args compiles the argument list for the QEMU command.
func (c *Command) Args() Arguments {
	a := Arguments{
		ArgKernel(c.Kernel),
		ArgInitrd(c.Initramfs),
	}
	if c.Machine != "" {
		a.Add(ArgMachine(c.Machine))
	}
	if c.CPU != "" {
		a.Add(ArgCPU(c.CPU))
	}
	if c.SMP != 0 {
		a.Add(ArgSMP(int(c.SMP)))
	}
	if c.Memory != 0 {
		a.Add(ArgMemory(int(c.Memory)))
	}
	if !c.NoKVM {
		a.Add(UniqueArg("enable-kvm"))
	}

	fdpath := func(i int) string {
		return fmt.Sprintf("/dev/fd/%d", i)
	}

	var addConsoleArgs func(int)
	switch c.TransportType {
	case TransportTypeISA:
		addConsoleArgs = func(fd int) {
			a.Add(ArgSerial("file:" + fdpath(fd)))
		}
	case TransportTypePCI:
		a.Add(ArgDevice("virtio-serial-pci", "max_ports=8"))
		addConsoleArgs = func(fd int) {
			vcon := fmt.Sprintf("vcon%d", fd)
			a.Add(
				ArgChardev("file", "id="+vcon, "path="+fdpath(fd)),
				ArgDevice("virtconsole", "chardev="+vcon),
			)
		}
	case TransportTypeMMIO:
		a.Add(ArgDevice("virtio-serial-device", "max_ports=8"))
		addConsoleArgs = func(fd int) {
			vcon := fmt.Sprintf("vcon%d", fd)
			a.Add(
				ArgChardev("file", "id="+vcon, "path="+fdpath(fd)),
				ArgDevice("virtconsole", "chardev="+vcon),
			)
		}
	}

	// Add stdout console.
	addConsoleArgs(1)

	// Write console output to file descriptors. Those are provided by the
	// [exec.Cmd.ExtraFiles].
	for idx := range c.AdditionalConsoles {
		// FDs 0, 1, 2 are standard in, out, err, so start at 3.
		addConsoleArgs(3 + idx)
	}

	a.Add(c.ExtraArgs...)
	a.Add(ArgAppend(c.kernelCmdlineArgs()...))

	return a
}

// kernelCmdlineArgs reruns the kernel cmdline arguments.
func (c *Command) kernelCmdlineArgs() []string {
	cmdline := []string{
		"console=" + c.TransportType.ConsoleDeviceName(0),
		"panic=-1",
	}
	if !c.Verbose {
		cmdline = append(cmdline, "loglevel=0")
	}
	if len(c.InitArgs) > 0 {
		cmdline = append(cmdline, "--")
		cmdline = append(cmdline, c.InitArgs...)
	}
	return cmdline
}

// Run the QEMU command with the given context.
//
// The final QEMU command is constructed, console processors are setup and the
// command is executed. A return code is returned. It can only be 0 if the
// guest system correctly communicated a 0 value via stdout. In any other case,
// a non 0 value is returned. If no error is returned, the value was received
// by the guest system.
func (c *Command) Run(ctx context.Context, stdout, stderr io.Writer) (int, error) {
	args, err := c.Args().Build()
	if err != nil {
		return 1, err
	}

	cmd := exec.CommandContext(ctx, c.Executable, args...)
	cmd.Stderr = stderr

	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		return 1, fmt.Errorf("get cmd stdout: %v", err)
	}

	// Collect processors so they can be easily closed.
	processors := make([]*consoleProcessor, 0, 16)
	closeProcessors := func() {
		for _, p := range processors {
			_ = p.Close()
		}
	}
	defer closeProcessors()

	// Create console output processors that fix line endings by stripping "\r".
	// Append the write end of the console processor pipe as extra file, so it
	// is present as additional file descriptor which can be used with the
	// "file" backend for QEMU console devices. [consoleProcessor.run] reads
	// from the read end of the pipe, cleans the output and writes it into
	// the actual target file on the host.
	processorsGroup := errgroup.Group{}
	for _, console := range c.AdditionalConsoles {
		p := consoleProcessor{Path: console}
		w, err := p.create()
		if err != nil {
			return 1, fmt.Errorf("create processor %s: %v", p.Path, err)
		}
		cmd.ExtraFiles = append(cmd.ExtraFiles, w)
		processors = append(processors, &p)
		processorsGroup.Go(p.run)
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
	// at program termination.
	rc, rcErr := ParseStdout(outPipe, stdout, c.Verbose)

	// Collect process information.
	if err := cmd.Wait(); err != nil {
		rc = 1
		return rc, fmt.Errorf("qemu: %v", err)
	}

	// Close console processors, so possible errors can be collected.
	closeProcessors()

	if err := processorsGroup.Wait(); err != nil {
		return 1, fmt.Errorf("processor error: %v", err)
	}

	return rc, rcErr
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
