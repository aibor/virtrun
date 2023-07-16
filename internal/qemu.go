package internal

import (
	"bufio"
	"bytes"
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
}

func (q *QEMUCommand) Cmd() *exec.Cmd {
	cmd := exec.Command(q.Binary, q.Args()...)
	return cmd
}

func (q *QEMUCommand) Args() []string {
	args := []string{
		"-kernel", q.Kernel,
		"-initrd", q.Initrd,
		"-machine", q.Machine,
		"-cpu", q.CPU,
		"-m", fmt.Sprintf("%d", q.Memory),
		"-serial", "stdio",
		"-display", "none",
		"-no-reboot",
	}

	if !q.NoKVM {
		args = append(args, "-enable-kvm")
	}

	for _, serialFile := range q.SerialFiles {
		args = append(args, "-serial", fmt.Sprintf("file:%s", serialFile))
	}

	cmdline := []string{
		"root=/dev/ram0",
		"console=ttyAMA0",
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

type Output struct {
	OutReader io.Reader
	ErrReader io.Reader
	OutWriter io.Writer
	ErrWriter io.Writer
}

// StartReaders starts a goroutine each for the given readers.
//
// They terminate if the readers are closed, a kernel panic message is read or
// the [RCFmt] is found. The returned read only channel of size one is used to
// communicate the read return code of the go test.
func Consume(output *Output) (*sync.WaitGroup, <-chan int, error) {
	panicRE, err := regexp.Compile(`^\[[0-9. ]+\] Kernel panic - not syncing: `)
	if err != nil {
		return nil, nil, fmt.Errorf("compile panic regex: %v", err)
	}

	rcStream := make(chan int, 1)
	readGroup := sync.WaitGroup{}

	readGroup.Add(1)
	go func() {
		defer readGroup.Done()
		defer close(rcStream)
		scanner := bufio.NewScanner(output.OutReader)
		for scanner.Scan() {
			line := scanner.Text()
			if panicRE.MatchString(line) {
				rcStream <- 255
				return
			}
			var rc int
			if _, err := fmt.Sscanf(line, RCFmt, &rc); err == nil {
				rcStream <- rc
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
