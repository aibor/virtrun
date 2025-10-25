// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

// Package main provides the main virtrun entry point.
package main

import (
	"os"

	"github.com/aibor/virtrun/internal/cmd"
)

func main() {
	os.Exit(cmd.Run(os.Args, os.Stdin, os.Stdout, os.Stderr))
}
