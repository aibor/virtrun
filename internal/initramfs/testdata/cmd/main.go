// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package main

//go:generate mkdir -vp ../lib
//go:generate $CC ../src/func1.c -shared -fPIC -nostdlib -o ../lib/libfunc1.so
//go:generate $CC ../src/func2.c -shared -fPIC -nostdlib -o ../lib/libfunc2.so
//go:generate $CC ../src/func3.c -shared -fPIC -nostdlib -L../lib -Wl,-rpath,${DOLLAR}ORIGIN/../lib -lfunc1 -o ../lib/libfunc3.so

//go:generate go build -trimpath -buildvcs=false -o ../bin/main .

// #cgo CFLAGS: -I${SRCDIR}/../src
// #cgo LDFLAGS: -L${SRCDIR}/../lib -Wl,-rpath,$ORIGIN/../lib -lfunc2 -lfunc3
// #include <defs.h>
import "C"
import "os"

func main() {
	// Return value is:
	// hexadecimal:  0x49
	// decimal:      73
	// octal:        0111
	os.Exit(int(C.func2() | C.func3()))
}
