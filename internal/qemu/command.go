package qemu

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"

	"golang.org/x/sync/errgroup"
)

var CommandPresets = map[string]Command{
	"amd64": {
		Binary:  "qemu-system-x86_64",
		Machine: "microvm",
		//Machine: "q35",
	},
	"arm64": {
		Binary:  "qemu-system-aarch64",
		Machine: "virt",
	},
}

type TransportType int

const (
	TransportTypeISA TransportType = iota
	TransportTypePCI
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
	// Stdout of the QEMU command.
	OutWriter io.Writer
	// Stderr of the QEMU command.
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

	cmd.CPU = "max"
	cmd.Memory = 256
	cmd.SMP = 2
	cmd.NoKVM = true
	cmd.TransportType = TransportTypeMMIO

	if runtime.GOARCH == arch {
		f, err := os.OpenFile("/dev/kvm", os.O_WRONLY, 0)
		_ = f.Close()
		if err == nil {
			cmd.NoKVM = false
		}
	}

	return &cmd, nil
}

// Output returns [Command.OutWriter] if set or [os.Stdout] otherwise.
func (q *Command) Output() io.Writer {
	if q.OutWriter == nil {
		return os.Stdout
	}
	return q.OutWriter
}

// ConsoleDeviceName returns the name of the console device in the guest.
func (q *Command) ConsoleDeviceName(num uint8) string {
	f := "hvc%d"
	if q.TransportType == TransportTypeISA {
		f = "ttyS%d"
	}
	return fmt.Sprintf(f, num)
}

// Validate checks for known incompatibilities.
func (q *Command) Validate() error {
	switch q.Machine {
	case "microvm":
		switch {
		case q.TransportType == TransportTypePCI:
			return fmt.Errorf("microvm does not support pci transport.")
		case q.TransportType == TransportTypeISA && len(q.ExtraFiles) > 0:
			msg := "microvm supports only one isa serial port, used for stdio."
			return fmt.Errorf(msg)
		}
	case "virt":
		if q.TransportType == TransportTypeISA {
			return fmt.Errorf("virt requires virtio-mmio")
		}
	case "q35", "pc":
		if q.TransportType == TransportTypeMMIO {
			return fmt.Errorf("%s does not work with virtio-mmio", q.Machine)
		}

	}
	return nil
}

// Output returns [Command.ErrWriter] if set or [os.Stderr] otherwise.
func (q *Command) ErrOutput() io.Writer {
	if q.ErrWriter == nil {
		return os.Stderr
	}
	return q.ErrWriter
}

// Cmd compiles the complete QEMU command.
func (q *Command) Cmd(ctx context.Context) *exec.Cmd {
	cmd := exec.CommandContext(ctx, q.Binary, q.Args()...)
	return cmd
}

// Args compiles the argument string for the QEMU command.
func (q *Command) Args() []string {
	a := args{
		argKernel(q.Kernel),
		argInitrd(q.Initrd),
		argMachine(q.Machine),
		argCPU(q.CPU),
		argSMP(int(q.SMP)),
		argMemory(int(q.Memory)),
		argDisplay("none"),
		argMonitor("none"),
		uniqueArg("no-reboot"),
		uniqueArg("nodefaults"),
		uniqueArg("no-user-config"),
	}

	if !q.NoKVM {
		a = append(a, uniqueArg("enable-kvm"))
	}

	fdpath := func(i int) string {
		return fmt.Sprintf("/dev/fd/%d", i)
	}

	var addConsoleArgs func(int)
	switch q.TransportType {
	case TransportTypeISA:
		addConsoleArgs = func(fd int) {
			a = append(a, argSerial("file:"+fdpath(fd)))
		}
	case TransportTypePCI:
		a = append(a, argDevice("virtio-serial-pci", "max_ports=8"))
		addConsoleArgs = func(fd int) {
			vcon := fmt.Sprintf("vcon%d", fd)
			a = append(a,
				argChardev("file", "id="+vcon, "path="+fdpath(fd)),
				argDevice("virtconsole", "chardev="+vcon),
			)
		}
	case TransportTypeMMIO:
		a = append(a, argDevice("virtio-serial-device", "max_ports=8"))
		addConsoleArgs = func(fd int) {
			vcon := fmt.Sprintf("vcon%d", fd)
			a = append(a,
				argChardev("file", "id="+vcon, "path="+fdpath(fd)),
				argDevice("virtconsole", "chardev="+vcon),
			)
		}
	}

	addConsoleArgs(1)

	// Write console output to file descriptors. Those are provided by the
	// [exec.Cmd.ExtraFiles].
	for idx := range q.ExtraFiles {
		// FDs 0, 1, 2 are standard in, out, err, so start at 3.
		addConsoleArgs(3 + idx)
	}

	cmdline := []string{
		"console=ttyAMA0",
		"console=" + q.ConsoleDeviceName(0),
		"panic=-1",
	}

	if !q.Verbose {
		cmdline = append(cmdline, "loglevel=0")
	}

	if len(q.InitArgs) > 0 {
		cmdline = append(cmdline, "--")
		cmdline = append(cmdline, q.InitArgs...)
	}

	a = append(a, argAppend(cmdline...))
	out, _ := a.build()
	return out
}

// Run the QEMU command with the given context.
func (q *Command) Run(ctx context.Context) (int, error) {
	rc := 1

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	rcParser := NewStdoutProcessor(q.Output(), q.Verbose)
	defer rcParser.Close()

	processorGroup, ctx := errgroup.WithContext(ctx)
	cmd := q.Cmd(ctx)
	cmd.Stdout = rcParser
	cmd.Stderr = q.ErrOutput()
	if q.Verbose {
		fmt.Println(cmd.String())
	}

	for _, extraFile := range q.ExtraFiles {
		p, err := NewSerialConsoleProcessor(extraFile)
		if err != nil {
			return rc, fmt.Errorf("serial processor %s: %v", extraFile, err)
		}
		defer p.Close()
		cmd.ExtraFiles = append(cmd.ExtraFiles, p.Writer())
		processorGroup.Go(p.Run)
	}

	processorGroup.Go(rcParser.Run)
	if err := cmd.Run(); err != nil {
		return rc, fmt.Errorf("run qemu: %v", err)
	}

	_ = rcParser.Close()
	for _, f := range cmd.ExtraFiles {
		_ = f.Close()
	}
	err := processorGroup.Wait()

	if rcParser.FoundRC {
		rc = rcParser.RC
	}
	return rc, err
}
