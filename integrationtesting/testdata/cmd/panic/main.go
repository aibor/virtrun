// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

//go:generate env CGO_ENABLED=0 go build -o ../../bin/ .
package main

import (
	"fmt"
	"os"
)

func run() error {
	if err := os.WriteFile("/proc/sys/kernel/sysrq", []byte("1"), 0755); err != nil {
		return err
	}
	return os.WriteFile("/proc/sysrq-trigger", []byte("c"), 0755)
}

func main() {
	if err := run(); err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
}
