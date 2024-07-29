// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

//go:generate env CGO_ENABLED=0 go build -o ../../bin/ .
package main

import (
	"fmt"
	"os"
	"strconv"
)

func run() error {
	var grow []byte

	memMB, err := strconv.Atoi(os.Args[1])
	if err != nil {
		panic("invalid input")
	}

	for i := 0; i < memMB; i++ {
		grow = append(grow, make([]byte, 1024*1024)...)
	}

	return fmt.Errorf("still alive :(")
}

func main() {
	if err := run(); err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
}
