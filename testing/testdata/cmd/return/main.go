// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

//go:generate env CGO_ENABLED=0 go build -o ../../bin/ .
package main

import (
	"os"
	"strconv"
)

func main() {
	// Use first argument as exit code.
	rc, _ := strconv.Atoi(os.Args[1])
	os.Exit(rc)
}
