// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package virtrun

import (
	"fmt"
	"io/fs"
	"os"
	"slices"

	"github.com/aibor/virtrun/internal/initramfs"
	"github.com/aibor/virtrun/internal/sys"
)

const (
	dataDir    = "data"
	libsDir    = "lib"
	modulesDir = "lib/modules"
)

type Initramfs struct {
	Arch           sys.Arch
	Binary         FilePath
	Files          FilePathList
	Modules        FilePathList
	StandaloneInit bool
	Keep           bool
}

type InitramfsArchive struct {
	Path string
	keep bool
}

func NewInitramfsArchive(cfg Initramfs) (*InitramfsArchive, error) {
	irfs := initramfs.New()

	err := buildFS(irfs, cfg)
	if err != nil {
		return nil, fmt.Errorf("build: %w", err)
	}

	path, err := WriteFSToTempFile(irfs, "")
	if err != nil {
		return nil, fmt.Errorf("write archive file: %w", err)
	}

	a := &InitramfsArchive{
		Path: path,
		keep: cfg.Keep,
	}

	return a, nil
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

func buildFS(f initramfs.FSAdder, cfg Initramfs) error {
	builder := fsBuilder{f}

	err := builder.addFilePathAs("main", string(cfg.Binary))
	if err != nil {
		return err
	}

	err = builder.addInit(cfg.Arch, cfg.StandaloneInit)
	if err != nil {
		return err
	}

	err = builder.addFilesTo(dataDir, cfg.Files, baseName)
	if err != nil {
		return err
	}

	err = builder.addFilesTo(modulesDir, cfg.Modules, modName)
	if err != nil {
		return err
	}

	binaryFiles := []string{string(cfg.Binary)}
	binaryFiles = append(binaryFiles, cfg.Files...)

	libs, err := sys.CollectLibsFor(binaryFiles...)
	if err != nil {
		return fmt.Errorf("collect libs: %w", err)
	}

	err = builder.addFilesTo(libsDir, slices.Collect(libs.Libs()), baseName)
	if err != nil {
		return err
	}

	err = builder.symlinkTo(libsDir, slices.Collect(libs.SearchPaths()))
	if err != nil {
		return err
	}

	return nil
}

// WriteFSToTempFile writes the given [fs.FS] as CPIO archive into a new
// temporary file in the given directory.
//
// It returns the path to the created file. If tmpDir is the empty string the
// default directory is used as returned by [os.TempDir].
//
// The caller is responsible for removing the file once it is not needed
// anymore.
func WriteFSToTempFile(fsys fs.FS, tmpDir string) (string, error) {
	file, err := os.CreateTemp(tmpDir, "initramfs")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer file.Close()

	writer := initramfs.NewCPIOFileWriter(file)
	defer writer.Close()

	err = initramfs.WriteFS(fsys, writer)
	if err != nil {
		_ = os.Remove(file.Name())
		return "", fmt.Errorf("create archive: %w", err)
	}

	return file.Name(), nil
}
