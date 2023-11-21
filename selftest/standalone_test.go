//go:build standalone

package sysinit_test

import (
	"testing"

	"github.com/aibor/virtrun/sysinit"
)

func TestMain(m *testing.M) {
	sysinit.RunTests(m)
}
