package qemu

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"golang.org/x/sync/errgroup"
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
	SMP uint8
	// Memory for the machine in MB.
	Memory uint16
	// Disable KVM support.
	NoKVM bool
	// Disable Virtio-MMIO support. Serial consoles depend on it. If this is set
	// to true, the legacy isa-pci bus will be used.
	NoVirtioMMIO bool
	// Print qemu command before running, increase guest kernel logging and
	// do not stop prinitng stdout when our RC string is found.
	Verbose bool
	// Additional serial files beside the default one used for stdout. They
	// will be present in the guest system as "/dev/ttySx" where x is the index
	// of the slice + 1.
	SerialFiles []string
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
	cmd := Command{
		CPU:    "max",
		Memory: 256,
		SMP:    2,
		NoKVM:  true,
	}

	switch arch {
	case "amd64":
		cmd.Binary = "qemu-system-x86_64"
		cmd.Machine = "microvm"
	case "arm64":
		cmd.Binary = "qemu-system-aarch64"
		cmd.Machine = "virt"
	default:
		return nil, fmt.Errorf("arch not supported: %s", arch)
	}

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

// SerialDevice returns the name of the serial console device in the guest.
func (q *Command) SerialDevice(num uint8) string {
	f := "hvc%d"
	if q.NoVirtioMMIO {
		f = "ttyS%d"
	}
	return fmt.Sprintf(f, num)
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
	args := []string{
		"-kernel", q.Kernel,
		"-initrd", q.Initrd,
		"-machine", q.Machine,
		"-cpu", q.CPU,
		"-smp", fmt.Sprintf("%d", q.SMP),
		"-m", fmt.Sprintf("%d", q.Memory),
		"-no-reboot",
		"-display", "none",
		"-monitor", "none",
		"-nodefaults",
		"-no-user-config",
	}

	if q.NoVirtioMMIO {
		args = append(args, "-serial", "stdio")
	} else {
		args = append(args,
			"-device", "virtio-serial-device",
			"-chardev", "stdio,id=virtiocon0",
			"-device", "virtconsole,chardev=virtiocon0",
		)
	}

	if !q.NoKVM {
		args = append(args, "-enable-kvm")
	}

	// Write serial console output to file descriptors. Those are provided by
	// the [exec.Cmd.ExtraFiles].
	for idx := range q.SerialFiles {
		// FDs 0, 1, 2 are standard in, out, err, so start at 3.
		path := fmt.Sprintf("/dev/fd/%d", 3+idx)

		if q.NoVirtioMMIO {
			args = append(args, "-serial", fmt.Sprintf("file:%s", path))
		} else {
			// virtiocon0 is stdio so start at 1.
			vcon := fmt.Sprintf("virtiocon%d", 1+idx)
			args = append(args,
				"-chardev", fmt.Sprintf("file,id=%s,path=%s", vcon, path),
				"-device", fmt.Sprintf("virtconsole,chardev=%s", vcon),
			)
		}
	}

	cmdline := []string{
		"console=ttyAMA0",
		"console=" + q.SerialDevice(0),
		"panic=-1",
	}

	if !q.Verbose {
		cmdline = append(cmdline, "loglevel=0")
	}

	if len(q.InitArgs) > 0 {
		cmdline = append(cmdline, "--")
		cmdline = append(cmdline, q.InitArgs...)
	}

	return append(args, "-append", strings.Join(cmdline, " "))
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

	for _, serialFile := range q.SerialFiles {
		p, err := NewSerialConsoleProcessor(serialFile)
		if err != nil {
			return rc, fmt.Errorf("serial processor %s: %v", serialFile, err)
		}
		defer p.Close()
		cmd.ExtraFiles = append(cmd.ExtraFiles, p.writePipe)
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
