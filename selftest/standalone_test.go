//go:build standalone

package selftest

import (
	"testing"

	"github.com/aibor/pidonetest/sysinit"
)

func TestMain(m *testing.M) {
	sysinit.RunTests(m)
}
