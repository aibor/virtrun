package pidonetest

import (
	"fmt"
	"os"
	"testing"

	"github.com/aibor/pidonetest/sysinit"
)

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
	if !sysinit.IsPidOne() {
		fmt.Printf("Error: %v\n", sysinit.NotPidOneError)
		os.Exit(127)
	}

	var err error
	defer sysinit.Poweroff(&err)

	err = sysinit.MountAll()
	if err != nil {
		err = fmt.Errorf("mounting file systems: %v", err)
		return
	}

	sysinit.PrintRC(m.Run())
}
