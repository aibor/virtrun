package pidonetest

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"testing"

	"github.com/aibor/go-pidonetest/internal"
)

// NotPidOneError is returned if the process does not have PID 1.
var NotPidOneError = errors.New("process has not PID 1")

// PrintPidOneTestRC prints the magic string communicating the return code of
// the tests.
func PrintPidOneTestRC(ret int) {
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

// Run sets up the system, runs the tests and shuts down the system.
//
// Call it from your `TestMain` function. It wraps [testing.M.Run] and returns
// only in case of failure. It is an error if the process does not run with
// PID 1, since the intention of this library is to run test binaries in an
// isolated system.
//
// Instead of using this convenience wrapper, you can run the commands yourself,
// of course. Make sure to call [PrintPidOneTestRC] with the output of your
// [testing.M.Run] call in order to communicate the result to the wrapping
// "pidonetest" CLI tool.
func Run(m *testing.M) {
	if !IsPidOne() {
		fmt.Printf("Error: %v", NotPidOneError)
		return
	}
	defer Poweroff()

	if err := MountAll(); err != nil {
		fmt.Printf("Error mounting file systems: %v", err)
		return
	}

	PrintPidOneTestRC(m.Run())
}
