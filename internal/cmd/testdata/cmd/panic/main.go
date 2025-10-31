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

	err := os.MkdirAll("/proc", os.ModePerm)
	if err != nil {
		panic("mkdir: " + err.Error())
	}

	err = syscall.Mount("proc", "/proc", "proc", 0, "")
	if err != nil {
		panic("mount: " + err.Error())
	}

	err = os.WriteFile("/proc/sysrq-trigger", []byte{'c'}, 0)
	if err != nil {
		panic("trigger panic: " + err.Error())
	}
}
