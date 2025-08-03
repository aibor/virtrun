// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"fmt"
	"os"
	"strconv"
)

func main() {
	// Use first argument as exit code.
	exitCode, err := strconv.Atoi(os.Args[1])
	if err != nil {
		panic("invalid input")
	}

	fmt.Fprintln(os.Stdout, "exit code:", exitCode)

	os.Exit(exitCode)
}
