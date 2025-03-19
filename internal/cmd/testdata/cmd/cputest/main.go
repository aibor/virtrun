// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"os"
	"runtime"
	"strconv"
)

func main() {
	// Use first argument as expected number of CPUs.
	expectedNumCPU, err := strconv.Atoi(os.Args[1])
	if err != nil {
		panic("invalid input")
	}

	if expectedNumCPU != runtime.NumCPU() {
		os.Exit(1)
	}
}
