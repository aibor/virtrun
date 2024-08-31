// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"os"
)

func main() {
	if os.Getpid() != 1 {
		panic("not PID 1")
	}

	if err := os.WriteFile("/proc/sys/kernel/sysrq", []byte{'1'}, 0); err != nil {
		panic("enable sysrq: " + err.Error())
	}

	if err := os.WriteFile("/proc/sysrq-trigger", []byte{'c'}, 0); err != nil {
		panic("trigger panic: " + err.Error())
	}
}
