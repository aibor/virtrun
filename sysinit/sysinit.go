package sysinit

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"github.com/aibor/pidonetest/internal"
)

// NotPidOneError is returned if the process does not have PID 1.
var NotPidOneError = errors.New("process has not PID 1")

// PrintRC prints the magic string communicating the return code of
// the tests.
func PrintRC(ret int) {
	fmt.Printf(internal.RCFmt, ret)
}

// IsPidOne returns true if the running process has PID 1.
func IsPidOne() bool {
	return os.Getpid() == 1
}

// IsPidOneChild returns true if the running process is a child of the process
// with PID 1.
func IsPidOneChild() bool {
	return os.Getppid() == 1
}

// Poweroff shuts down the system.
//
// Call when done, or deferred right at the beginning of your `TestMain`
// function.
func Poweroff() {
	// Silence the kernel so it does not show up in our test output.
	_ = os.WriteFile("/proc/sys/kernel/printk", []byte("0"), 0755)

	if err := syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF); err != nil {
		fmt.Printf("error calling power off: %v\n", err)
	}
}
