package sysinit

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// RCFmt is the format string for communicating the test results
//
// It is parsed in the qemu wrapper. Not present in the output if the test
// binary panicked.
const RCFmt = "INIT_RC: %d\n"

// ErrNotPidOne may be returned if the process is expected to be run as PID 1
// but is not.
var ErrNotPidOne = errors.New("process does not have ID 1")

// PrintRC prints the magic string communicating the return code of
// the tests.
func PrintRC(ret int) {
	fmt.Printf(RCFmt, ret)
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
// function. If the given error pointer and error are not nil, print it before
// shutting down.
func Poweroff(err *error) {
	if err != nil && *err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", *err)
	}

	// Silence the kernel so it does not show up in our test output.
	_ = os.WriteFile("/proc/sys/kernel/printk", []byte("0"), 0755)

	if err := syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF); err != nil {
		fmt.Printf("error calling power off: %v\n", err)
	}
	// We just told the system to shutdown. There's no point in staying around.
	os.Exit(0)
}

// Run is the entry point for an actual init system. It prepares the system
// to be used. Preparing steps are:
// - Guarding itself to be actually PID 1.
// - Setup system poweroff on its exit.
// - Mount all known virtual system file systems.
//
// Once this is done, the given function is run. The function must not call
// [os.Exit] itself since the program would not be able to ensure a correct
// system termination.
//
// After that, a return code is sent to stdout for consumption by the host
// process. The return code returned by the function is used, unless it
// returned with an error. If the error is an [exec.ExitError], it is
// parsed and its return code is used. Otherwise the return code is 99.
func Run(fn func() (int, error)) error {
	if !IsPidOne() {
		return ErrNotPidOne
	}

	// From here on we can assume we are a systems's init program. termination
	// will lead to system shutdown, or kernel panic, if we do not shutdown
	// correctly.
	var err error
	defer Poweroff(&err)

	if err = ConfigureLoopbackInterface(); err != nil {
		return err
	}

	if err = MountAll(); err != nil {
		return err
	}

	rc, err := fn()
	var eerr *exec.ExitError
	if err != nil {
		if errors.As(err, &eerr) {
			rc = eerr.ExitCode()
			err = nil
		} else {
			rc = 127
		}
	}
	PrintRC(rc)

	return nil
}
