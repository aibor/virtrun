package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aibor/virtrun/sysinit"
)

func main() {
	err := sysinit.Run(func() (int, error) {
		pattern := os.Args[1]
		if pattern == "" {
			return 98, fmt.Errorf("empty pattern")
		}

		paths, err := filepath.Glob(os.Args[1])
		if err != nil {
			return 97, err
		}

		return 0, sysinit.ExecParallel(paths, os.Args[2:], os.Stdout, os.Stderr)
	})
	if err != nil {
		// Usually, we never reach this point, if the Run function reached the
		// poweroff command. So, it is only used if the guarding returns an
		// error.
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
