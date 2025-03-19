// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FilePath string

func (f *FilePath) String() string {
	return string(*f)
}

func (f *FilePath) Set(s string) error {
	path, err := AbsoluteFilePath(s)

	*f = FilePath(path)

	return err
}

type FilePathList []string

func (f *FilePathList) String() string {
	return strings.Join(*f, ",")
}

func (f *FilePathList) Set(s string) error {
	for _, e := range strings.Split(s, ",") {
		path, err := AbsoluteFilePath(e)
		if err != nil {
			return err
		}

		*f = append(*f, path)
	}

	return nil
}

func AbsoluteFilePath(path string) (string, error) {
	if path == "" {
		return "", ErrEmptyFilePath
	}

	path, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("absolute path: %w", err)
	}

	return path, nil
}

func MustAbsoluteFilePath(path string) string {
	abs, err := AbsoluteFilePath(path)
	if err != nil {
		panic(err)
	}

	return abs
}

func ValidateFilePath(name string) error {
	stat, err := os.Stat(name)
	if err != nil {
		return err //nolint:wrapcheck
	}

	if !stat.Mode().IsRegular() {
		return ErrNotRegularFile
	}

	return nil
}
