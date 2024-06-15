// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package sysinit

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/sys/unix"
)

// LoadModules loads all files found in the given directory as kernel modules.
func LoadModules(dir string) error {
	files, err := ListRegularFiles(dir)
	if err != nil {
		return fmt.Errorf("list module files: %w", err)
	}

	for _, file := range files {
		err := LoadModule(file, "")
		if err != nil {
			return fmt.Errorf("load module %s: %w", file, err)
		}
	}

	return nil
}

// LoadModule loads the kernel module located at the given path with the given
// parameters.
//
// The file may be compressed. The caller is responsible to ensure the module
// belongs to the running kernel and all dependencies are satisfied.
func LoadModule(path string, params string) error {
	flags := 0
	if hasFinitCompressionExtension(filepath.Base(path)) {
		flags |= unix.MODULE_INIT_COMPRESSED_FILE
	}

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	return finitModule(int(file.Fd()), params, flags)
}

func finitModule(fd int, params string, flags int) error {
	err := unix.FinitModule(fd, params, flags)
	if err != nil {
		return fmt.Errorf("finit: %w", err)
	}

	return nil
}

func hasFinitCompressionExtension(fileName string) bool {
	fileNameParts := strings.Split(fileName, ".")
	extension := fileNameParts[len(fileNameParts)-1]

	return isFinitCompressionExtension(extension)
}

// isFinitCompressionExtension checks if the given extension is one of the
// known extensions finit_module(2) supports.
func isFinitCompressionExtension(extension string) bool {
	supportedExtensions := []string{"gz", "xz", "zst"}

	return slices.Contains(supportedExtensions, extension)
}
