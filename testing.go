package virtrun

import (
	"fmt"
	"os"
	"testing"
)

// Tests sets up the system, runs the tests and shuts down the system.
//
// Call it from your `TestMain` function. It wraps [testing.M.Run] and returns
// only in case of failure. It is an error if the process does not run with
// PID 1, since the intention of this library is to run test binaries in an
// isolated system.
func Tests(m *testing.M) {
	err := Init(func() (int, error) {
		return m.Run(), nil
	})
	rc := 1
	if err == ErrNotPidOne {
		rc = 127

	}
	fmt.Fprintf(os.Stderr, "Error: %v", err)
	os.Exit(rc)
}
