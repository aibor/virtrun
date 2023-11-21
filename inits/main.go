package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aibor/virtrun/internal/initramfs"
	"github.com/aibor/virtrun/sysinit"
)

func runInit() (int, error) {
	err := sysinit.Run(func() (int, error) {
		dir := initramfs.FilesDir
		files, err := os.ReadDir(dir)
		if err != nil {
			return 98, err
		}

		paths := make([]string, len(files))
		for idx, f := range files {
			paths[idx] = filepath.Join(dir, f.Name())
		}

		return 0, sysinit.ExecParallel(paths, os.Args[1:], os.Stdout, os.Stderr)
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
