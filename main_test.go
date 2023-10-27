//go:build pidonetest

package pidonetest_test

import (
	"testing"

	"github.com/aibor/pidonetest"
)

func TestMain(m *testing.M) {
	pidonetest.Run(m)
}
