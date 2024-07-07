// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"os"

	"github.com/aibor/virtrun/internal/initprog"
	"github.com/aibor/virtrun/internal/initramfs"
)

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

type initramfsArchive struct {
	path string
	keep bool
}

func (a *initramfsArchive) Close() error {
	if a.keep {
		fmt.Fprintf(os.Stderr, "initramfs kept at: %s\n", a.path)

		return nil
	}

	return os.Remove(a.path)
}

func newInitramfsArchive(args initramfsArgs, arch string) (*initramfsArchive, error) {
	irfs, err := newInitramfs(string(args.binary), args.standalone, arch)
	if err != nil {
		return nil, fmt.Errorf("new: %v", err)
	}

	err = irfs.AddFiles("data", args.files...)
	if err != nil {
		return nil, fmt.Errorf("add files: %v", err)
	}

	err = irfs.AddRequiredSharedObjects()
	if err != nil {
		return nil, fmt.Errorf("add libs: %v", err)
	}

	err = irfs.AddFiles("lib/modules", args.modules...)
	if err != nil {
		return nil, fmt.Errorf("add modules: %v", err)
	}

	path, err := irfs.WriteToTempFile("")
	if err != nil {
		return nil, fmt.Errorf("write to temp file: %v", err)
	}

	a := &initramfsArchive{
		path: path,
		keep: args.keepInitramfs,
	}

	return a, nil
}
