package qemu

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"golang.org/x/sync/errgroup"
)

// TransportType represents QEMU IO transport types.
type TransportType int

const (
	// ISA legacy transport. Should work for amd64 in any case. With "microvm"
	// machine type only provides one console for stdout.
	TransportTypeISA TransportType = iota
	// Virtio PCI transport. Requires kernel built with CONFIG_VIRTIO_PCI.
	TransportTypePCI
	// Virtio MMIO transport. Requires kernel built with CONFIG_VIRTIO_MMIO.
	TransportTypeMMIO
)

// Command is a single QEMU command that can be run.
type Command struct {
	// Path to the qemu-system binary
	Binary string
	// Path to the kernel to boot. The kernel should have Virtio-MMIO support
	// compiled in. If not, set the NoVirtioMMIO flag.
	Kernel string
	// Path to the initrd to boot with.
	Initrd string
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
	// Print qemu command before running, increase guest kernel logging and
	// do not stop prinitng stdout when our RC string is found.
	Verbose bool
	// ExtraArgs are  extra arguments that are passed to the QEMU command.
	// They may not interfere with the essential arguments set by the command
	// itself or an error wil lbe returned on [Command.Run].
	ExtraArgs Arguments
	// Additional files attached to consoles besides the default one used for
	// stdout. They will be present in the guest system as "/dev/ttySx" or
	// "/dev/hvcx" where x is the index of the slice + 1.
	ExtraFiles []string
	// Arguments to pass to the init binary.
	InitArgs []string
	// Stdout of the QEMU command. If not set, os.Stdout will be used.
	OutWriter io.Writer
	// Stderr of the QEMU command. If not set, os.Stderr will be used.
	ErrWriter io.Writer
}

// output returns [Command.OutWriter] if set or [os.Stdout] otherwise.
func (c *Command) output() io.Writer {
	if c.OutWriter == nil {
		return os.Stdout
	}
	return c.OutWriter
}

// errOutput returns [Command.ErrWriter] if set or [os.Stderr] otherwise.
func (c *Command) errOutput() io.Writer {
	if c.ErrWriter == nil {
		return os.Stderr
	}
	return c.ErrWriter
}

// ConsoleDeviceName returns the name of the console device in the guest.
func (c *Command) ConsoleDeviceName(num uint8) string {
	f := "hvc%d"
	if c.TransportType == TransportTypeISA {
		f = "ttyS%d"
	}
	return fmt.Sprintf(f, num)
}

// AddExtraFile adds an additional file to the QEMU command. This will be
// writable from the guest via the device name returned by this command.
// Console device number is starting at 1, as console 0 is the default stdout.
func (c *Command) AddExtraFile(file string) string {
	c.ExtraFiles = append(c.ExtraFiles, file)
	return c.ConsoleDeviceName(uint8(len(c.ExtraFiles)))
}

// Validate checks for known incompatibilities.
func (c *Command) Validate() error {
	switch c.Machine {
	case "microvm":
		switch {
		case c.TransportType == TransportTypePCI:
			return fmt.Errorf("microvm does not support pci transport.")
		case c.TransportType == TransportTypeISA && len(c.ExtraFiles) > 0:
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

// args compiles the argument string for the QEMU command.
func (c *Command) args() Arguments {
	a := Arguments{
		ArgKernel(c.Kernel),
		ArgInitrd(c.Initrd),
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
	for idx := range c.ExtraFiles {
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
		"console=" + c.ConsoleDeviceName(0),
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

// cmd constructs a new [exec.Cmd] with the given context and arguments.
//
// Besides the [exec.Cmd] it returns a list of [ConsoleProcessor]s that depend
// on the number of extra files/consoles added to the command. They are
// required to process  the console output and write it to the expected files
// on the host. They are only created but not started.
func (c *Command) cmd(ctx context.Context, args []string) (*exec.Cmd, []*consoleProcessor) {
	cmd := exec.CommandContext(ctx, c.Binary, args...)
	cmd.Stderr = c.errOutput()
	if c.Verbose {
		fmt.Println(cmd.String())
	}

	procs := make([]*consoleProcessor, 0, 8)
	for _, extraFile := range c.ExtraFiles {
		p := consoleProcessor{Path: extraFile}
		procs = append(procs, &p)
	}

	return cmd, procs
}

// Run the QEMU command with the given context.
//
// The final QEMU command is constructed, console processors are setup and the
// command is executed. A return code is returned. It can only be 0 if the
// guest system correctly communicated a 0 value via stdout. In any other case,
// a non 0 value is returned. If no error is returned, the value was received
// by the guest system.
func (c *Command) Run(ctx context.Context) (int, error) {
	rc := 1

	args, err := c.args().Build()
	if err != nil {
		return rc, err
	}

	cmd, processors := c.cmd(ctx, args)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return rc, err
	}

	processorsGroup := errgroup.Group{}
	for _, p := range processors {
		w, err := p.create()
		if err != nil {
			return rc, fmt.Errorf("create processor %s: %v", p.Path, err)
		}
		defer p.close()
		cmd.ExtraFiles = append(cmd.ExtraFiles, w)
		processorsGroup.Go(p.run)
	}

	if err := cmd.Start(); err != nil {
		return rc, fmt.Errorf("start: %v", err)
	}

	rc, rcErr := ParseStdout(out, c.output(), c.Verbose)

	if err := cmd.Wait(); err != nil {
		// Never return 0 along with an execution error.
		if rc == 0 {
			rc = 1
		}
		return rc, fmt.Errorf("wait: %v", err)
	}

	for _, p := range processors {
		_ = p.close()
	}
	proccessorsErr := processorsGroup.Wait()

	if rcErr != nil {
		// rc == 0 is fine here since the error is an processor error only.
		return rc, rcErr
	}
	return rc, proccessorsErr
}
