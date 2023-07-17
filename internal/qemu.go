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
	"strings"
	"sync"
)

// RCFmt is the format string for communicating the test results
//
// It is parsed in the qemu wrapper. Not present in the output if the test
// binary panicked.
const RCFmt = "GO_PIDONETEST_RC: %d\n"

// QEMUCommand is a single QEMU command that can be run.
type QEMUCommand struct {
	Binary      string
	Kernel      string
	Initrd      string
	Machine     string
	CPU         string
	Memory      uint16
	NoKVM       bool
	SerialFiles []string
	InitArgs    []string
	OutWriter   io.Writer
	ErrWriter   io.Writer
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
		"console=ttyS0",
		"panic=-1",
		"quiet",
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
}

type RCValue struct {
	Found bool
	RC    int
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
		for scanner.Scan() {
			line := scanner.Text()
			if panicRE.MatchString(line) {
				rcStream <- RCValue{Found: false, RC: 126}
				return
			}
			var rc int
			if _, err := fmt.Sscanf(line, RCFmt, &rc); err == nil {
				rcStream <- RCValue{Found: true, RC: rc}
				return
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
