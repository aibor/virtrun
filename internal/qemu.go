package internal

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

// QEMUCommand is a single QEMU command that can be run.
type QEMUCommand struct {
	Binary      string
	Kernel      string
	Initrd      string
	Machine     string
	CPU         string
	SMP         uint8
	Memory      uint16
	NoKVM       bool
	Verbose     bool
	SerialFiles []string
	InitArgs    []string
	OutWriter   io.Writer
	ErrWriter   io.Writer
}

// NewQEMUCommand creates a new QEMUCommand with defaults set to the
// given architecture. If it does not match the host architecture, the
// [QEMUCommand.NoKVM] flag ist set. Supported architectures so far:
// amd64, arm64.
func NewQEMUCommand(arch string) (*QEMUCommand, error) {
	qemuCmd := QEMUCommand{
		CPU:    "max",
		Memory: 256,
		SMP:    2,
		NoKVM:  true,
	}

	switch arch {
	case "amd64":
		qemuCmd.Binary = "qemu-system-x86_64"
		qemuCmd.Machine = "microvm,pit=off,pic=off,isa-serial=off,rtc=off"
	case "arm64":
		qemuCmd.Binary = "qemu-system-aarch64"
		qemuCmd.Machine = "virt"
	default:
		return nil, fmt.Errorf("arch not supported: %s", arch)
	}

	if runtime.GOARCH == arch {
		f, err := os.OpenFile("/dev/kvm", os.O_WRONLY, 0)
		_ = f.Close()
		if err == nil {
			qemuCmd.NoKVM = false
		}
	}

	return &qemuCmd, nil
}

// Output returns [QEMUCommand.OutWriter] if set or [os.Stdout] otherwise.
func (q *QEMUCommand) Output() io.Writer {
	if q.OutWriter == nil {
		return os.Stdout
	}
	return q.OutWriter
}

// Output returns [QEMUCommand.ErrWriter] if set or [os.Stderr] otherwise.
func (q *QEMUCommand) ErrOutput() io.Writer {
	if q.ErrWriter == nil {
		return os.Stderr
	}
	return q.ErrWriter
}

// Cmd compiles the complete QEMU command.
func (q *QEMUCommand) Cmd(ctx context.Context) *exec.Cmd {
	cmd := exec.CommandContext(ctx, q.Binary, q.Args()...)
	return cmd
}

// Args compiles the argument string for the QEMU command.
func (q *QEMUCommand) Args() []string {
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
		"-nographic",
		"-nodefaults",
		"-no-user-config",
		"-device", "virtio-serial-device",
		"-chardev", "stdio,id=virtiocon0",
		"-device", "virtconsole,chardev=virtiocon0",
	}

	if !q.NoKVM {
		args = append(args, "-enable-kvm")
	}

	// Write serial console output to file descriptors. Those are provided by
	// the [exec.Cmd.ExtraFiles].
	for idx := range q.SerialFiles {
		// virtiocon0 is stdio so start at 1.
		vcon := fmt.Sprintf("virtiocon%d", 1+idx)
		// FDs 0, 1, 2 are standard in, out, err, so start at 3.
		path := fmt.Sprintf("/dev/fd/%d", 3+idx)

		args = append(
			args,
			"-chardev", fmt.Sprintf("file,id=%s,path=%s", vcon, path),
			"-device", fmt.Sprintf("virtconsole,chardev=%s", vcon),
		)
	}

	cmdline := []string{
		"console=ttyAMA0",
		"console=hvc0",
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
func (q *QEMUCommand) Run(ctx context.Context) (int, error) {
	rc := 1

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	rcParser := NewRCParser(q.Output(), q.Verbose)
	defer rcParser.Close()

	processorGroup, ctx := errgroup.WithContext(ctx)
	cmd := q.Cmd(ctx)
	cmd.Stdout = rcParser
	cmd.Stderr = q.ErrOutput()
	if q.Verbose {
		fmt.Println(cmd.String())
	}

	for _, serialFile := range q.SerialFiles {
		p, err := NewSerialProcessor(serialFile)
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
