// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package internal

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aibor/virtrun/internal/initprog"
	"github.com/aibor/virtrun/internal/initramfs"
)

type InitramfsArgs struct {
	Arch          string
	Binary        FilePath
	Files         FilePathList
	Modules       FilePathList
	Standalone    bool
	KeepInitramfs bool
}

func NewInitramfsArchive(args InitramfsArgs) (*InitramfsArchive, error) {
	irfs, err := newInitramfs(string(args.Binary), args.Standalone, args.Arch)
	if err != nil {
		return nil, fmt.Errorf("new: %v", err)
	}

	err = irfs.AddFiles("data", args.Files...)
	if err != nil {
		return nil, fmt.Errorf("add files: %v", err)
	}

	err = irfs.AddRequiredSharedObjects()
	if err != nil {
		return nil, fmt.Errorf("add libs: %v", err)
	}

	for idx, module := range args.Modules {
		name := fmt.Sprintf("%04d-%s", idx, filepath.Base(module))

		err = irfs.AddFile("lib/modules", name, module)
		if err != nil {
			return nil, fmt.Errorf("add modules: %v", err)
		}
	}

	path, err := irfs.WriteToTempFile("")
	if err != nil {
		return nil, fmt.Errorf("write to temp file: %v", err)
	}

	a := &InitramfsArchive{
		Path: path,
		keep: args.KeepInitramfs,
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

	return os.Remove(a.Path)
}

func newInitramfs(
	mainBinary string,
	standalone bool,
	arch string,
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
	arch string,
) (*initramfs.Initramfs, error) {
	// In the default wrapped mode a pre-compiled init is used that just
	// executes "/main".
	init, err := initprog.For(arch)
	if err != nil {
		return nil, fmt.Errorf("embedded init: %v", err)
	}

	irfs := initramfs.New(initramfs.WithVirtualInitFile(init))

	err = irfs.AddFile("/", "main", mainBinary)
	if err != nil {
		return nil, fmt.Errorf("add main file: %v", err)
	}

	return irfs, nil
}
