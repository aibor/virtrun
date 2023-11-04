package main

import (
	"fmt"
	"os"

	"github.com/aibor/pidonetest/internal/initramfs"
	"github.com/aibor/pidonetest/sysinit"
)

func main() {
	if err := sysinit.Run(initramfs.FilesDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
