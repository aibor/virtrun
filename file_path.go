// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type filePath string

func (f filePath) MarshalText() ([]byte, error) {
	return []byte(f), nil
}

func (f *filePath) UnmarshalText(text []byte) error {
	var err error
	*f, err = absoluteFilePath(string(text))

	return err
}

var errNotRegularFile = errors.New("not a regular file")

func (f filePath) check() error {
	stat, err := os.Stat(string(f))
	if err != nil {
		return err
	}

	if !stat.Mode().IsRegular() {
		return errNotRegularFile
	}

	return nil
}

type filePathList []string

func (f filePathList) String() string {
	return strings.Join(f, ",")
}

func (f *filePathList) Set(value string) error {
	path, err := absoluteFilePath(value)
	if err != nil {
		return err
	}

	*f = append(*f, string(path))

	return nil
}

var errEmptyFilePath = errors.New("file path must not be empty")

func absoluteFilePath(path string) (filePath, error) {
	if path == "" {
		return "", errEmptyFilePath
	}

	path, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("ensure absolute path: %v", err)
	}

	return filePath(path), nil
}
