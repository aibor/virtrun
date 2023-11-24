package main

import (
	"fmt"
	"os"

	"github.com/aibor/virtrun/sysinit"
)

func runInit() (int, error) {
	err := sysinit.Run(func() (int, error) {
		return 0, sysinit.Exec("/main", os.Args[1:], os.Stdout, os.Stderr)
	})
	if err == sysinit.ErrNotPidOne {
		return 127, err
	}
	return 126, err
}

func main() {
	rc, err := runInit()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
	os.Exit(rc)
}
