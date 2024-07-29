// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

//go:generate env CGO_ENABLED=0 go build -o ../../bin/ .
package main

import (
	"fmt"
	"os"
)

func writeFile(path string, data string) error {
	return os.WriteFile(path, []byte(data), os.ModePerm)
}

func run() error {
	if os.Getpid() != 1 {
		return fmt.Errorf("not PID 1")
	}

	if err := writeFile("/proc/sys/kernel/sysrq", "1"); err != nil {
		return fmt.Errorf("enable sysrq: %w", err)
	}

	if err := writeFile("/proc/sysrq-trigger", "c"); err != nil {
		return fmt.Errorf("trigger panic: %w", err)
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
}
