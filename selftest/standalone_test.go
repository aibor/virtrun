//go:build standalone

package selftest

import (
	"testing"

	"github.com/aibor/virtrun"
)

func TestMain(m *testing.M) {
	virtrun.Tests(m)
}
