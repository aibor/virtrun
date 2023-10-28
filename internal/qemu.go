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
	"sync"
)

// RCFmt is the format string for communicating the test results
//
// It is parsed in the qemu wrapper. Not present in the output if the test
// binary panicked.
const RCFmt = "INIT_RC: %d\n"

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
func (q *QEMUCommand) Cmd() *exec.Cmd {
	cmd := exec.Command(q.Binary, q.Args()...)
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
		cmdline = append(cmdline, "quiet")
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
	cmd := q.Cmd()

	cmdOut, err := cmd.StdoutPipe()
	if err != nil {
		return rc, fmt.Errorf("get command stdout: %v", err)
	}
	defer cmdOut.Close()

	cmdErr, err := cmd.StderrPipe()
	if err != nil {
		return rc, fmt.Errorf("get command stderr: %v", err)
	}
	defer cmdErr.Close()

	if err := cmd.Start(); err != nil {
		return rc, fmt.Errorf("run qemu: %v", err)
	}
	p := cmd.Process
	if p != nil {
		defer func() {
			_ = p.Kill()
		}()
	}

	readGroup, rcStream, err := Consume(&Output{
		OutReader: cmdOut,
		ErrReader: cmdErr,
		OutWriter: q.Output(),
		ErrWriter: q.ErrOutput(),
		Verbose:   q.Verbose,
	})
	if err != nil {
		return rc, fmt.Errorf("start readers: %v", err)
	}

	done := make(chan bool)
	go func() {
		_ = cmd.Wait()
		readGroup.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return rc, ctx.Err()
	case r := <-rcStream:
		if r.Found || r.RC != 0 {
			return r.RC, nil
		}
		return rc, nil
	case <-done:
		return rc, nil
	}
}

type Output struct {
	OutReader io.Reader
	ErrReader io.Reader
	OutWriter io.Writer
	ErrWriter io.Writer
	Verbose   bool
}

type RCValue struct {
	Found bool
	RC    int

	sent bool
}

// StartReaders starts a goroutine each for the given readers.
//
// They terminate if the readers are closed, a kernel panic message is read or
// the [RCFmt] is found. The returned read only channel of size one is used to
// communicate the read return code of the go test.
func Consume(output *Output) (*sync.WaitGroup, <-chan RCValue, error) {
	panicRE, err := regexp.Compile(`^\[[0-9. ]+\] Kernel panic - not syncing: `)
	if err != nil {
		return nil, nil, fmt.Errorf("compile panic regex: %v", err)
	}

	rcStream := make(chan RCValue, 1)
	readGroup := sync.WaitGroup{}

	readGroup.Add(1)
	go func() {
		defer readGroup.Done()
		defer close(rcStream)
		scanner := bufio.NewScanner(output.OutReader)
		var rc RCValue
		for scanner.Scan() {
			line := scanner.Text()
			if panicRE.MatchString(line) {
				if !rc.sent {
					rc.RC = 126
				}
			} else if _, err := fmt.Sscanf(line, RCFmt, &rc.RC); err == nil {
				if !rc.sent {
					rc.Found = true
				}
			}
			if !rc.sent && rc != (RCValue{}) {
				rcStream <- rc
				rc.sent = true
				if !output.Verbose {
					return
				}
			}
			fmt.Fprintln(output.OutWriter, line)
		}
	}()

	readGroup.Add(1)
	go func() {
		defer readGroup.Done()
		_, _ = io.Copy(output.ErrWriter, output.ErrReader)
	}()

	return &readGroup, rcStream, nil
}
