// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package internal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

var ErrNotRegularFile = errors.New("not a regular file")

func (f FilePath) check() error {
	stat, err := os.Stat(string(f))
	if err != nil {
		return err
	}

	if !stat.Mode().IsRegular() {
		return ErrNotRegularFile
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

var ErrEmptyFilePath = errors.New("file path must not be empty")

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
