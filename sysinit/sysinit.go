package sysinit

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/aibor/pidonetest/internal"
	"golang.org/x/sync/errgroup"
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
}

// Exec executes the given file wit the given arguments. Output and errors are
// written to the given writers immediately. Might return [exec.ExitError].
func Exec(path string, args []string, outWriter, errWriter io.Writer) error {
	cmd := exec.Command(path, args...)
	cmd.Stdout = outWriter
	cmd.Stderr = errWriter
	return cmd.Run()
}

// ExecParallel executes the given files in parallel. Each is called with the
// given args. Output of the commands is written to the given out and err
// writers once the command exited. Might return [exec.ExitError].
func ExecParallel(paths []string, args []string, outW, errW io.Writer) error {
	if !IsPidOne() {
		return NotPidOneError
	}

	var (
		writers   sync.WaitGroup
		outStream = make(chan []byte)
		errStream = make(chan []byte)
		addWriter = func(writer io.Writer, byteStream <-chan []byte) {
			writers.Add(1)
			go func(w io.Writer, r <-chan []byte) {
				defer writers.Done()
				for b := range r {
					fmt.Fprint(w, string(b))
				}
			}(writer, byteStream)
		}
	)

	addWriter(outW, outStream)
	addWriter(errW, errStream)

	eg := errgroup.Group{}
	for _, path := range paths {
		path := path
		eg.Go(func() error {
			var outBuf, errBuf bytes.Buffer
			err := Exec(path, args, &outBuf, &errBuf)
			outStream <- outBuf.Bytes()
			errStream <- errBuf.Bytes()
			return err
		})
	}

	err := eg.Wait()
	close(outStream)
	close(errStream)
	writers.Wait()

	return err
}
