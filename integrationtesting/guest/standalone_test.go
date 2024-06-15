// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

//go:build standalone

package main_test

import (
	"testing"

	"github.com/aibor/virtrun/sysinit"
)

func TestMain(m *testing.M) {
	sysinit.RunTests(m)
}
