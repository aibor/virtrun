// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"os"
	"strconv"
)

const maxByte = 256

func main() {
	// Use first argument as output length.
	length, err := strconv.Atoi(os.Args[1])
	if err != nil {
		panic("invalid input")
	}

	// Use second argument as number of repeated lines.
	repeat, err := strconv.Atoi(os.Args[2])
	if err != nil {
		panic("invalid input")
	}

	// Write incrementing byte values repeatedly.
	output := make([]byte, length)
	for i := range output {
		output[i] = byte(i % maxByte)
	}

	for i := range repeat {
		if i > 0 {
			_, _ = os.Stdout.Write([]byte{'\n'})
		}
		// time.Sleep(time.Millisecond)
		_, _ = os.Stdout.Write(output)
	}
}
