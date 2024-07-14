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
	var grow []byte
	memMB, _ := strconv.Atoi(os.Args[1])

	for i := 0; i < memMB; i++ {
		grow = append(grow, make([]byte, 1024*1024)...)
	}
}
