// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"os"
	"syscall"
)

func main() {
	if os.Getpid() != 1 {
		panic("not PID 1")
	}

	if err := os.MkdirAll("/proc", os.ModePerm); err != nil {
		panic("mkdir: " + err.Error())
	}

	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		panic("mount: " + err.Error())
	}

	if err := os.WriteFile("/proc/sysrq-trigger", []byte{'c'}, 0); err != nil {
		panic("trigger panic: " + err.Error())
	}
}
