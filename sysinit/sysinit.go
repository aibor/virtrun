package sysinit

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/aibor/virtrun/internal/qemu"
)

// ErrNotPidOne may be returned if the process is expected to be run as PID 1
// but is not.
var ErrNotPidOne = errors.New("process does not have ID 1")

// PrintRC prints the magic string communicating the return code of
// the tests.
func PrintRC(ret int) {
	fmt.Printf(qemu.RCFmt, ret)
}

// PrintErrorAndRC examines the given error, prints it and sets the return code
// to the given errRC. If there is no error, the given rc is printed.
func PrintErrorAndRC(err error, errRC, rc int) {
	// Always print the error before printing the RC, since output
	// processing stops once RC line is found and we want to make sure the
	// error can be seen by the user.
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		// Always return a non zero return code in case of error.
		rc = errRC
	}
	PrintRC(rc)
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
// parsed and its return code is used. Otherwise the return code is 127 in case
// it was never set or 126 in case there was an error.
func Run(fn func() (int, error)) error {
	if !IsPidOne() {
		return ErrNotPidOne
	}

	// From here on we can assume we are a systems's init program. Termination
	// will lead to system shutdown, or kernel panic, if we do not shutdown
	// correctly.
	defer Poweroff()

	var (
		// Set fallthrough rc to non zero value, so it must be set to zero
		// explicitly by the callers function later.
		rc      = 127 // Fallthrough return code.
		errRC   = 126 // Return code that is used in case of errors.
		err     error
		exitErr *exec.ExitError
	)

	// Setup the error and return code printing so it is always printed. In
	// case of setup errors, the failure is communicated properly as well.
	defer func() {
		PrintErrorAndRC(err, errRC, rc)
	}()

	// Setup the system.
	if err = ConfigureLoopbackInterface(); err != nil {
		return err
	}
	if err = MountAll(); err != nil {
		return err
	}
	if err = CreateCommonSymlinks(); err != nil {
		return err
	}

	// Run callers function. The returned rc is irrelevant if any error is
	// returned, because the deferred error handling will override it.
	rc, err = fn()
	if errors.As(err, &exitErr) {
		rc = exitErr.ExitCode()
		err = nil
	}

	return err
}
