// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"os"
	"strings"

	"github.com/aibor/virtrun/internal/sys"
)

// FilePath is an absolute path of an existing regular file.
type FilePath string

func (f *FilePath) String() string {
	return string(*f)
}

// Set sets [FilePath] to the given path, if valid.
func (f *FilePath) Set(input string) error {
	path, err := sys.AbsolutePath(input)
	if err != nil {
		return err //nolint:wrapcheck
	}

	*f = FilePath(path)

	return nil
}

// FilePathList is a list of absolute paths of existing regular files.
type FilePathList []string

func (f *FilePathList) String() string {
	return strings.Join(*f, ",")
}

// Set adds the given file path to the list, if valid. An empty string clears
// the list.
func (f *FilePathList) Set(input string) error {
	if input == "" {
		*f = nil
		return nil
	}

	for e := range strings.SplitSeq(input, ",") {
		path, err := sys.AbsolutePath(e)
		if err != nil {
			return err //nolint:wrapcheck
		}

		*f = append(*f, path)
	}

	return nil
}

// ValidateFilePath validates that the file with the given name is an existing
// regular file.
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
