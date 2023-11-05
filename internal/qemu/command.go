package qemu

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"

	"golang.org/x/sync/errgroup"
)

var CommandPresets = map[string]Command{
	"amd64": {
		Binary:        "qemu-system-x86_64",
		Machine:       "microvm",
		TransportType: TransportTypeMMIO,
		CPU:           "max",
	},
	"arm64": {
		Binary:        "qemu-system-aarch64",
		Machine:       "virt",
		TransportType: TransportTypeMMIO,
		CPU:           "max",
	},
}

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

// NewCommand creates a new [Command] with defaults set to the given
// architecture. If it does not match the host architecture, the
// [Command.NoKVM] flag ist set. Supported architectures so far: amd64, arm64.
func NewCommand(arch string) (*Command, error) {
	cmd, exists := CommandPresets[arch]
	if !exists {
		return nil, fmt.Errorf("arch not supported: %s", arch)
	}

	cmd.Memory = 256
	cmd.SMP = 2
	cmd.NoKVM = !KVMAvailableFor(arch)

	return &cmd, nil
}

// Output returns [Command.OutWriter] if set or [os.Stdout] otherwise.
func (c *Command) Output() io.Writer {
	if c.OutWriter == nil {
		return os.Stdout
	}
	return c.OutWriter
}

// Output returns [Command.ErrWriter] if set or [os.Stderr] otherwise.
func (c *Command) ErrOutput() io.Writer {
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
// Console device number is starting at 2, as console 0 and 1 are stdout and
// stderr.
func (c *Command) AddExtraFile(file string) string {
	c.ExtraFiles = append(c.ExtraFiles, file)
	return c.ConsoleDeviceName(uint8(len(c.ExtraFiles) + 1))
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

// Args compiles the argument string for the QEMU command.
func (c *Command) Args() Arguments {
	a := Arguments{
		ArgKernel(c.Kernel),
		ArgInitrd(c.Initrd),
		ArgMachine(c.Machine),
		ArgCPU(c.CPU),
		ArgSMP(int(c.SMP)),
		ArgMemory(int(c.Memory)),
		ArgDisplay("none"),
		ArgMonitor("none"),
		UniqueArg("no-reboot"),
		UniqueArg("nodefaults"),
		UniqueArg("no-user-config"),
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

	// Add stdout and stderr fd.
	addConsoleArgs(1)
	addConsoleArgs(2)

	// Write console output to file descriptors. Those are provided by the
	// [exec.Cmd.ExtraFiles].
	for idx := range c.ExtraFiles {
		// FDs 0, 1, 2 are standard in, out, err, so start at 3.
		addConsoleArgs(3 + idx)
	}

	a.Add(ArgAppend(c.KernelCmdlineArgs()...))

	return a
}

// KernelCmdlineArgs reruns the kernel cmdline arguments.
func (c *Command) KernelCmdlineArgs() []string {
	cmdline := []string{
		"console=ttyAMA0",
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

// Run the QEMU command with the given context.
func (c *Command) Cmd(ctx context.Context, args []string) (*exec.Cmd, []*ConsoleProcessor) {
	cmd := exec.CommandContext(ctx, c.Binary, args...)
	cmd.Stderr = c.ErrOutput()
	if c.Verbose {
		fmt.Println(cmd.String())
	}

	procs := make([]*ConsoleProcessor, 0, 8)
	for _, extraFile := range c.ExtraFiles {
		p := ConsoleProcessor{Path: extraFile}
		procs = append(procs, &p)
	}

	return cmd, procs
}

// Run the QEMU command with the given context.
func (c *Command) Run(ctx context.Context) (int, error) {
	rc := 1

	args, err := c.Args().Build()
	if err != nil {
		return rc, err
	}

	cmd, processors := c.Cmd(ctx, args)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return rc, err
	}

	processorsGroup := errgroup.Group{}
	for _, p := range processors {
		w, err := p.Create()
		if err != nil {
			return 1, fmt.Errorf("create processor %s: %v", p.Path, err)
		}
		defer p.Close()
		cmd.ExtraFiles = append(cmd.ExtraFiles, w)
		processorsGroup.Go(p.Run)
	}

	if err := cmd.Start(); err != nil {
		return rc, fmt.Errorf("start: %v", err)
	}

	rc, rcErr := ParseStdout(out, c.Output(), c.Verbose)

	if err := cmd.Wait(); err != nil {
		return rc, fmt.Errorf("wait: %v", err)
	}

	for _, p := range processors {
		_ = p.Close()
	}
	proccessorsErr := processorsGroup.Wait()

	return rc, errors.Join(rcErr, proccessorsErr)
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
