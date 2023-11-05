//go:build standalone

package selftest

import (
	"testing"

	"github.com/aibor/pidonetest"
)

func TestMain(m *testing.M) {
	pidonetest.RunTests(m)
}
