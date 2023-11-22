package virtrun

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"

	"golang.org/x/sync/errgroup"

	"github.com/aibor/virtrun/qemu"
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
	TransportType qemu.TransportType
	// ExtraArgs are  extra arguments that are passed to the QEMU command.
	// They may not interfere with the essential arguments set by the command
	// itself or an error will be returned on [Command.Run].
	ExtraArgs qemu.Arguments
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
		ExtraArgs: qemu.Arguments{
			qemu.ArgDisplay("none"),
			qemu.ArgMonitor("none"),
			qemu.UniqueArg("no-reboot"),
			qemu.UniqueArg("nodefaults"),
			qemu.UniqueArg("no-user-config"),
		},
	}
	switch arch {
	case "amd64":
		cmd.Executable = "qemu-system-x86_64"
		cmd.Machine = "q35"
		cmd.TransportType = qemu.TransportTypePCI
	case "arm64":
		cmd.Executable = "qemu-system-aarch64"
		cmd.Machine = "virt"
		cmd.TransportType = qemu.TransportTypeMMIO
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
		case c.TransportType == qemu.TransportTypePCI:
			return fmt.Errorf("microvm does not support pci transport.")
		case c.TransportType == qemu.TransportTypeISA && len(c.AdditionalConsoles) > 0:
			msg := "microvm supports only one isa serial port, used for stdio."
			return fmt.Errorf(msg)
		}
	case "virt":
		if c.TransportType == qemu.TransportTypeISA {
			return fmt.Errorf("virt requires virtio-mmio")
		}
	case "q35", "pc":
		if c.TransportType == qemu.TransportTypeMMIO {
			return fmt.Errorf("%s does not work with virtio-mmio", c.Machine)
		}

	}
	return nil
}

// Args compiles the argument list for the QEMU command.
func (c *Command) Args() qemu.Arguments {
	a := qemu.Arguments{
		qemu.ArgKernel(c.Kernel),
		qemu.ArgInitrd(c.Initramfs),
	}
	if c.Machine != "" {
		a.Add(qemu.ArgMachine(c.Machine))
	}
	if c.CPU != "" {
		a.Add(qemu.ArgCPU(c.CPU))
	}
	if c.SMP != 0 {
		a.Add(qemu.ArgSMP(int(c.SMP)))
	}
	if c.Memory != 0 {
		a.Add(qemu.ArgMemory(int(c.Memory)))
	}
	if !c.NoKVM {
		a.Add(qemu.UniqueArg("enable-kvm"))
	}

	fdpath := func(i int) string {
		return fmt.Sprintf("/dev/fd/%d", i)
	}

	var addConsoleArgs func(int)
	switch c.TransportType {
	case qemu.TransportTypeISA:
		addConsoleArgs = func(fd int) {
			a.Add(qemu.ArgSerial("file:" + fdpath(fd)))
		}
	case qemu.TransportTypePCI:
		a.Add(qemu.ArgDevice("virtio-serial-pci", "max_ports=8"))
		addConsoleArgs = func(fd int) {
			vcon := fmt.Sprintf("vcon%d", fd)
			a.Add(
				qemu.ArgChardev("file", "id="+vcon, "path="+fdpath(fd)),
				qemu.ArgDevice("virtconsole", "chardev="+vcon),
			)
		}
	case qemu.TransportTypeMMIO:
		a.Add(qemu.ArgDevice("virtio-serial-device", "max_ports=8"))
		addConsoleArgs = func(fd int) {
			vcon := fmt.Sprintf("vcon%d", fd)
			a.Add(
				qemu.ArgChardev("file", "id="+vcon, "path="+fdpath(fd)),
				qemu.ArgDevice("virtconsole", "chardev="+vcon),
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
	a.Add(qemu.ArgAppend(c.kernelCmdlineArgs()...))

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
