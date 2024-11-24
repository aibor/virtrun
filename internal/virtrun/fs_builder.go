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

	"github.com/aibor/virtrun/internal/initramfs"
	"github.com/aibor/virtrun/internal/sys"
)

type nameFunc func(idx int, path string) string

func baseName(_ int, path string) string {
	return filepath.Base(path)
}

func modName(idx int, path string) string {
	return fmt.Sprintf("%04d-%s", idx, filepath.Base(path))
}

type fsBuilder struct {
	fs initramfs.FSAdder
}

func (b *fsBuilder) mkdirAll(dir string) error {
	return b.fs.MkdirAll(dir) //nolint:wrapcheck
}

func (b *fsBuilder) addFilePathAs(name, source string) error {
	return b.fs.Add(name, func() (fs.File, error) { //nolint:wrapcheck
		return os.Open(source)
	})
}

func (b *fsBuilder) addOpenFileAs(name string, file fs.File) error {
	return b.fs.Add(name, func() (fs.File, error) { //nolint:wrapcheck
		return file, nil
	})
}

func (b *fsBuilder) symlink(target, name string) error {
	return b.fs.Symlink(target, name) //nolint:wrapcheck
}

func (b *fsBuilder) addFilesTo(dir string, files []string, fn nameFunc) error {
	err := b.mkdirAll(dir)
	if err != nil {
		return err
	}

	for idx, path := range files {
		name := filepath.Join(dir, fn(idx, path))

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

		err := b.mkdirAll(filepath.Dir(path))
		if err != nil {
			return err
		}

		err = b.symlink(libsDir, path)
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *fsBuilder) addInit(arch sys.Arch, standalone bool) error {
	// In standalone mode, the main file is supposed to work as a complete init
	// matching our requirements.
	if standalone {
		return b.symlink("main", "init")
	}

	initFile, err := initProgFor(arch)
	if err != nil {
		return err
	}

	return b.addOpenFileAs("init", initFile)
}
