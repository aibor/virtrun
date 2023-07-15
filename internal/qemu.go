package internal

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

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
