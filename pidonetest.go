package pidonetest

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

// NotPidOneError is returned if the process does not have PID 1.
var NotPidOneError = errors.New("process has not PID 1")

func mountfs(path, fstype string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %v", path, err)
	}

	if err := syscall.Mount(fstype, path, fstype, 0, ""); err != nil {
		return fmt.Errorf("mount %s: %v", path, err)
	}

	return nil
}

func setupSystem() error {
	mounts := []struct {
		dest   string
		fstype string
	}{
		{"/dev", "devtmpfs"},
		{"/proc", "proc"},
		{"/run", "tmpfs"},
		{"/sys", "sysfs"},
		{"/sys/fs/bpf", "bpf"},
		{"/sys/kernel/tracing", "tracefs"},
		{"/tmp", "tmpfs"},
	}

	for _, mp := range mounts {
		if err := mountfs(mp.dest, mp.fstype); err != nil {
			return err
		}
		time.Sleep(1 * time.Millisecond)
	}

	// Silence the kernel so it does not show up in our test output.
	if err := os.WriteFile("/proc/sys/kernel/printk", []byte("0"), 0755); err != nil {
		return fmt.Errorf("set printk: %v", err)
	}

	return nil
}

func printTrace() error {
	f, err := os.ReadFile("/sys/kernel/tracing/trace")
	if err != nil {
		return fmt.Errorf("open trace pipe: %v", err)
	}

	log := make([]string, 0)
	for _, l := range strings.Split(strings.TrimSpace(string(f)), "\n") {
		if !strings.HasPrefix(l, "#") {
			log = append(log, l)
		}
	}

	if len(log) > 0 {
		fmt.Println("Kernel trace log:")
		fmt.Println(f)
	}

	return nil
}

func poweroff() {
	if err := syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF); err != nil {
		fmt.Printf("error calling power off: %v\n", err)
	}
}

// Run sets up the system, runs the tests and poweroffs the system.
//
// Call it from your `TestMain` function. It wraps [testing.M.Run] and returns
// the integer indicating if the tests passed or failed. It returns 255 and an
// error if the setup failed. It is an error if the process does not run with
// PID 1, since the intention of this library is to run test binaries in an
// isolates system.
func Run(m *testing.M) (int, error) {
	if os.Getpid() != 1 {
		return 255, NotPidOneError
	}

	defer poweroff()

	if err := setupSystem(); err != nil {
		return 255, fmt.Errorf("set up system: %v\n", err)
	}

	ret := m.Run()

	// Special magic line that is used to communicate the return code of the
	// tests. It is parsed in the qemu wrapper. Not present in the output if
	// the test binary panicked.
	fmt.Printf("GO_PIDONETEST_RC: %d\n", ret)

	// An error with printing the trace should not fail the tests, so
	// communicate an error but also the tests return code, so the caller can
	// decide how to handle the situation.
	var err error
	if testing.Verbose() {
		err = printTrace()
	}
	return ret, err
}
