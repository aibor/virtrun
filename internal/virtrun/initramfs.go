// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package virtrun

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aibor/virtrun/internal/initramfs"
	"github.com/aibor/virtrun/internal/sys"
)

type Initramfs struct {
	Arch           sys.Arch
	Binary         sys.FilePath
	Files          sys.FilePathList
	Modules        sys.FilePathList
	StandaloneInit bool
	Keep           bool
}

func NewInitramfsArchive(cfg Initramfs) (*InitramfsArchive, error) {
	irfs, err := newInitramfs(string(cfg.Binary), cfg.StandaloneInit, cfg.Arch)
	if err != nil {
		return nil, fmt.Errorf("new: %w", err)
	}

	err = irfs.AddFiles("data", cfg.Files...)
	if err != nil {
		return nil, fmt.Errorf("add files: %w", err)
	}

	err = irfs.AddRequiredSharedObjects()
	if err != nil {
		return nil, fmt.Errorf("add libs: %w", err)
	}

	for idx, module := range cfg.Modules {
		name := fmt.Sprintf("%04d-%s", idx, filepath.Base(module))

		err = irfs.AddFile("lib/modules", name, module)
		if err != nil {
			return nil, fmt.Errorf("add modules: %w", err)
		}
	}

	path, err := irfs.WriteToTempFile("")
	if err != nil {
		return nil, fmt.Errorf("write to temp file: %w", err)
	}

	a := &InitramfsArchive{
		Path: path,
		keep: cfg.Keep,
	}

	return a, nil
}

type InitramfsArchive struct {
	Path string
	keep bool
}

func (a *InitramfsArchive) Cleanup() error {
	if a.keep {
		fmt.Fprintf(os.Stderr, "initramfs kept at: %s\n", a.Path)
		return nil
	}

	err := os.Remove(a.Path)
	if err != nil {
		return fmt.Errorf("remove: %w", err)
	}

	return nil
}

func newInitramfs(
	mainBinary string,
	standalone bool,
	arch sys.Arch,
) (*initramfs.Initramfs, error) {
	// In standalone mode, the first file (which might be the only one)
	// is supposed to work as an init matching our requirements.
	if standalone {
		return initramfs.New(initramfs.WithRealInitFile(mainBinary)), nil
	}

	return newInitramfsWithInit(mainBinary, arch)
}

func newInitramfsWithInit(
	mainBinary string,
	arch sys.Arch,
) (*initramfs.Initramfs, error) {
	// In the default wrapped mode a pre-compiled init is used that just
	// executes "/main".
	init, err := initProgFor(arch)
	if err != nil {
		return nil, fmt.Errorf("embedded init: %w", err)
	}

	irfs := initramfs.New(initramfs.WithVirtualInitFile(init))

	err = irfs.AddFile("/", "main", mainBinary)
	if err != nil {
		return nil, fmt.Errorf("add main file: %w", err)
	}

	return irfs, nil
}
