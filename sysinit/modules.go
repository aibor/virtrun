// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package sysinit

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

// LoadModules loads all files found in the given directory as kernel modules.
func LoadModules(dir string) error {
	files, err := ListRegularFiles(dir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
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
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	// Try finit_module(2) first, as it is the more comfortable syscall. If it
	// is not available try again with init_module(2).
	err = finitModule(f, params)
	if err != nil {
		if errors.Is(err, errors.ErrUnsupported) {
			return initModule(f, params)
		}

		return err
	}

	return nil
}

func initModule(f *os.File, params string) error {
	decompressed, err := decompress(f)
	if err != nil {
		return fmt.Errorf("decompress: %w", err)
	}

	var data bytes.Buffer

	_, err = io.Copy(&data, decompressed)
	if err != nil {
		return fmt.Errorf("read file: %v", err)
	}

	err = unix.InitModule(data.Bytes(), params)
	if err != nil {
		return fmt.Errorf("init_module: %v", err)
	}

	return nil
}

type namedReader interface {
	io.Reader
	Name() string
}

func decompress(r namedReader) (io.Reader, error) {
	switch fileNameExtension(r.Name()) {
	case "ko":
		return r, nil
	case "gz":
		return gzip.NewReader(r)
	default:
		return nil, fmt.Errorf("unknown extension: %w", errors.ErrUnsupported)
	}
}

func finitModule(f *os.File, params string) error {
	flags := 0
	if hasFinitCompressionExtension(f.Name()) {
		flags |= unix.MODULE_INIT_COMPRESSED_FILE
	}

	err := unix.FinitModule(int(f.Fd()), params, flags)
	if err != nil {
		// If finit_module is not available, an EOPNOTSUPP is returned.
		if errors.Is(err, syscall.EOPNOTSUPP) {
			return fmt.Errorf("finit_module: %w", errors.ErrUnsupported)
		}

		return fmt.Errorf("finit_module: %v", err)
	}

	return nil
}

func fileNameExtension(fileName string) string {
	fileNameParts := strings.Split(fileName, ".")

	return fileNameParts[len(fileNameParts)-1]
}

func hasFinitCompressionExtension(fileName string) bool {
	extension := fileNameExtension(fileName)

	return isFinitCompressionExtension(extension)
}

// isFinitCompressionExtension checks if the given extension is one of the
// known extensions finit_module(2) supports.
func isFinitCompressionExtension(extension string) bool {
	supportedExtensions := []string{"gz", "xz", "zst"}

	return slices.Contains(supportedExtensions, extension)
}
