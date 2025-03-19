// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"os"
	"strings"

	"github.com/aibor/virtrun/internal/sys"
)

type FilePath string

func (f *FilePath) String() string {
	return string(*f)
}

func (f *FilePath) Set(s string) error {
	path, err := sys.AbsolutePath(s)
	if err != nil {
		return err //nolint:wrapcheck
	}

	*f = FilePath(path)

	return nil
}

type FilePathList []string

func (f *FilePathList) String() string {
	return strings.Join(*f, ",")
}

func (f *FilePathList) Set(s string) error {
	for e := range strings.SplitSeq(s, ",") {
		path, err := sys.AbsolutePath(e)
		if err != nil {
			return err //nolint:wrapcheck
		}

		*f = append(*f, path)
	}

	return nil
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
