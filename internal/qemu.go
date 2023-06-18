package internal

import (
	"fmt"
	"os/exec"
	"strings"
)

type QEMUCommand struct {
	Binary   string
	Kernel   string
	Initrd   string
	Machine  string
	CPU      string
	Memory   uint16
	NoKVM    bool
	TestArgs []string
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

	cmdline := []string{
		"root=/dev/ram0",
		"console=ttyAMA0",
		"console=ttyS0",
		"panic=-1",
		"quiet",
	}

	if len(q.TestArgs) > 0 {
		cmdline = append(cmdline, "--")
		cmdline = append(cmdline, q.TestArgs...)

	}

	return append(args, "-append", strings.Join(cmdline, " "))
}
