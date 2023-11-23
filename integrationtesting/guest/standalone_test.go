//go:build standalone

package main

import (
	"testing"

	"github.com/aibor/virtrun/sysinit"
)

func TestMain(m *testing.M) {
	sysinit.RunTests(m)
}
