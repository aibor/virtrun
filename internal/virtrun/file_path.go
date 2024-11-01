// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package virtrun

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aibor/virtrun/internal/sys"
)

type FilePath string

func (f FilePath) MarshalText() ([]byte, error) {
	return []byte(f), nil
}

func (f *FilePath) UnmarshalText(text []byte) error {
	var err error
	*f, err = AbsoluteFilePath(string(text))

	return err
}

func (f FilePath) Validate() error {
	stat, err := os.Stat(string(f))
	if err != nil {
		return err //nolint:wrapcheck
	}

	if !stat.Mode().IsRegular() {
		return ErrNotRegularFile
	}

	return nil
}

func (f FilePath) ValidateBinary(arch sys.Arch) error {
	file, err := os.Open(string(f))
	if err != nil {
		return err //nolint:wrapcheck
	}

	stat, err := file.Stat()
	if err != nil {
		return err //nolint:wrapcheck
	}

	if !stat.Mode().IsRegular() {
		return ErrNotRegularFile
	}

	err = ValidateELF(file, arch)
	if err != nil {
		return fmt.Errorf("elf header: %w", err)
	}

	return nil
}

type FilePathList []string

func (f FilePathList) String() string {
	return strings.Join(f, ",")
}

func (f *FilePathList) Set(value string) error {
	path, err := AbsoluteFilePath(value)
	if err != nil {
		return err
	}

	*f = append(*f, string(path))

	return nil
}

func AbsoluteFilePath(path string) (FilePath, error) {
	if path == "" {
		return "", ErrEmptyFilePath
	}

	path, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("ensure absolute path: %w", err)
	}

	return FilePath(path), nil
}
