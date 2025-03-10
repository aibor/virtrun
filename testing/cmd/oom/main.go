// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"os"
	"strconv"
)

const megaByte = 1024 * 1024

func main() {
	if os.Getppid() != 1 {
		panic("not child of PID 1")
	}

	var grow []byte

	memMB, err := strconv.Atoi(os.Args[1])
	if err != nil {
		panic("invalid input")
	}

	// Try to ensure the expected OOM task dump message is printed.
	_ = os.WriteFile("/proc/sys/vm/oom_dump_tasks", []byte("0"), 0)
	_ = os.WriteFile("/proc/sys/vm/panic_on_oom", []byte("0"), 0)
	_ = os.WriteFile("/proc/sys/kernel/printk", []byte("4"), 0)

	for range memMB {
		grow = append(grow, make([]byte, megaByte)...)
	}

	_ = grow

	panic("still alive :(")
}
