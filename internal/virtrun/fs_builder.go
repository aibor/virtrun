// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package virtrun

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/aibor/virtrun/internal/virtfs"
)

type nameFunc func(idx int, path string) string

func baseName(_ int, path string) string {
	return filepath.Base(path)
}

func modName(idx int, path string) string {
	return fmt.Sprintf("%04d-%s", idx, filepath.Base(path))
}

type fsBuilder struct {
	virtfs.FSAdder
}

func (b *fsBuilder) addFilePathAs(name, source string) error {
	return b.Add(name, func() (fs.File, error) {
		return os.Open(source)
	})
}

func (b *fsBuilder) addFilesTo(
	dir string,
	files []string,
	nameFn nameFunc,
) error {
	err := b.MkdirAll(dir)
	if err != nil {
		return err
	}

	for idx, path := range files {
		name := filepath.Join(dir, nameFn(idx, path))

		err := b.addFilePathAs(name, path)
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *fsBuilder) symlinkTo(dir string, paths []string) error {
	for _, path := range paths {
		if path == dir {
			continue
		}

		path := strings.TrimPrefix(path, string(filepath.Separator))
		if path == "" || path == dir {
			continue
		}

		err := b.MkdirAll(filepath.Dir(path))
		if err != nil {
			return err
		}

		err = b.Symlink(dir, path)
		if err != nil {
			return err
		}
	}

	return nil
}
