// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sys

import (
	"fmt"
	"path/filepath"
)

// AbsolutePath returns the absolute path as resolved by [filepath.Abs].
//
// It returns [ErrEmptyPath] if the given path is empty.
func AbsolutePath(path string) (string, error) {
	if path == "" {
		return "", ErrEmptyPath
	}

	path, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("absolute path: %w", err)
	}

	return path, nil
}

// MustAbsolutePath calls [AbsolutePath] and panics in case of errors.
func MustAbsolutePath(path string) string {
	abs, err := AbsolutePath(path)
	if err != nil {
		panic(err)
	}

	return abs
}
