package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aibor/virtrun/internal/initramfs"
)

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no init file given")
	}

	initFile, err := absPath(args[0])
	if err != nil {
		return err
	}

	additionalFiles := make([]string, 0)
	for _, file := range args[1:] {
		path, err := absPath(file)
		if err != nil {
			return err
		}
		additionalFiles = append(additionalFiles, path)
	}

	libSearchPath := os.Getenv("LD_LIBRARY_PATH")

	initRamFS := initramfs.New(initFile)
	if err := initRamFS.AddFiles(additionalFiles...); err != nil {
		return fmt.Errorf("add files: %v", err)
	}
	if err := initRamFS.ResolveLinkedLibs(libSearchPath); err != nil {
		return fmt.Errorf("add linked libs: %v", err)
	}
	if err := initRamFS.WriteCPIO(os.Stdout); err != nil {
		return fmt.Errorf("write: %v", err)
	}

	return nil
}

func absPath(file string) (string, error) {
	path, err := filepath.Abs(file)
	if err != nil {
		return "", fmt.Errorf("lookup absolute path for %s: %v", file, err)
	}
	return path, nil
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
