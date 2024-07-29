// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package main

import (
	"os"
	"strconv"
)

const megaByte = 1024 * 1024

func main() {
	var grow []byte

	memMB, err := strconv.Atoi(os.Args[1])
	if err != nil {
		panic("invalid input")
	}

	for range memMB {
		grow = append(grow, make([]byte, megaByte)...)
	}

	_ = grow

	panic("still alive :(")
}
