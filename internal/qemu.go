package internal

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

// RCFmt is the format string for communicating the test results
//
// It is parsed in the qemu wrapper. Not present in the output if the test
// binary panicked.
const RCFmt = "INIT_RC: %d\n"

var panicRE = regexp.MustCompile(`^\[[0-9. ]+\] Kernel panic - not syncing: `)

// QEMUCommand is a single QEMU command that can be run.
type QEMUCommand struct {
	Binary      string
	Kernel      string
	Initrd      string
	Machine     string
	CPU         string
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
		NoKVM:  true,
	}

	switch arch {
	case "amd64":
		qemuCmd.Binary = "qemu-system-x86_64"
		qemuCmd.Machine = "microvm"
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
		"-m", fmt.Sprintf("%d", q.Memory),
		"-no-reboot",
		"-serial", "stdio",
		"-display", "none",
		"-nodefaults",
		"-monitor", "none",
		"-no-user-config",
	}

	if !q.NoKVM {
		args = append(args, "-enable-kvm")
	}

	for _, serialFile := range q.SerialFiles {
		args = append(args, "-serial", fmt.Sprintf("file:%s", serialFile))
	}

	cmdline := []string{
		"console=ttyAMA0",
		"console=ttyS0",
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

// FixSerialFiles remove carriage returns from the [QEMUCommand.SerialFiles].
//
// The serial console ends files with "\r\n" but "go test" does not like the
// carriage returns. It reads the whole file and writes it back.
func (q *QEMUCommand) FixSerialFiles() error {
	for _, serialFile := range q.SerialFiles {
		content, err := os.ReadFile(serialFile)
		if err != nil {
			return fmt.Errorf("read serial file %s: %v", serialFile, err)
		}

		replaced := bytes.ReplaceAll(content, []byte("\r"), nil)
		if err := os.WriteFile(serialFile, replaced, 0644); err != nil {
			return fmt.Errorf("write serial file %s: %v", serialFile, err)
		}
	}

	return nil
}

// Run the QEMU command with the given context.
func (q *QEMUCommand) Run(ctx context.Context) (int, error) {
	rc := 1

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	rcParser := NewRCParser(q.Output(), q.Verbose)
	defer rcParser.Close()

	cmd := q.Cmd(ctx)
	cmd.Stdout = rcParser
	cmd.Stderr = q.ErrOutput()
	if q.Verbose {
		fmt.Println(cmd.String())
	}

	outDone := rcParser.Start()

	if err := cmd.Run(); err != nil {
		return rc, fmt.Errorf("run qemu: %v", err)
	}

	rcParser.Close()
	<-outDone

	if rcParser.FoundRC {
		rc = rcParser.RC
	}
	return rc, nil
}

// RCParser wraps [io.PipeWriter] and is used to find our well-known RC
// string for communication the return code from the guest. Call [RCParser.Close]
// in order to terminate the reader.
type RCParser struct {
	*io.PipeWriter
	scanner *bufio.Scanner
	output  io.Writer
	verbose bool
	RC      int
	FoundRC bool
}

// NewRCParser sets up a new RCParser.
func NewRCParser(output io.Writer, verbose bool) *RCParser {
	r, w := io.Pipe()
	return &RCParser{
		PipeWriter: w,
		scanner:    bufio.NewScanner(r),
		output:     output,
		verbose:    verbose,
	}
}

// Start the reader that writes into the given output writer.
//
// It starts a go routine that terminates when the RCParser is closed. The
// returned channel is closed when the reader processed all input.
func (p *RCParser) Start() <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		for p.scanner.Scan() {
			line := p.scanner.Text()
			if panicRE.MatchString(line) {
				if !p.FoundRC {
					p.RC = 126
				}
			} else if _, err := fmt.Sscanf(line, RCFmt, &p.RC); err == nil {
				p.FoundRC = true
			}
			if !p.FoundRC || p.verbose {
				fmt.Fprintln(p.output, line)
			}
		}
	}()
	return done
}
