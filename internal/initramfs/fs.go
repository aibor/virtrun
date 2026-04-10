// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package initramfs

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/aibor/virtrun/internal/sys"
	"github.com/aibor/virtrun/internal/virtfs"
)

const (
	dataDir    = "data"
	confDir    = "etc"
	libsDir    = "lib"
	modulesDir = "lib/modules"

	initPath   = "init"
	configPath = "etc/config.bin"
	hostsPath  = "etc/hosts"
	mainPath   = "main"

	hostsFile = `
127.0.0.1       localhost
::1             localhost ip6-localhost ip6-loopback
ff02::1         ip6-allnodes
ff02::2         ip6-allrouters
`
)

// Spec specifies the input for initramfs archive creation.
type Spec struct {
	// Executable is the main binary that is either executed directly or by the
	// init program depending on the presence of [Initramfs.Init].
	Executable string

	// Files is a list of any additional files that should be added to the
	// dataDir directory. For ELF files the required dynamic libraries are
	// added the libsDir directory.
	Files []string

	// Modules is a list of kernel module files. They are added to the
	// modulesDir directory.
	Modules []string

	// Fsys is the file system all files should be copied from.
	Fsys fs.FS

	// Init provides the init program. If not set, the [Initramfs.Executable] is
	// used as init program itself and expected to handle system setup and clean
	// shutdown.
	Init []byte

	// Config provides config data for the init program.
	Config []byte
}

func (i Spec) executables() []string {
	files := make([]string, 0, 1+len(i.Files))
	files = append(files, i.Executable)
	files = append(files, i.Files...)

	return files
}

// FS is a virtual file system tree that represents the initramfs.
type FS = virtfs.FS

// New creates a new initramfs CPIO archive file.
//
// The archive consists of a main executable that is either executed directly or
// by the init program. All other files are added to the dataDir directory.
// Kernel modules are added to modulesDir directory. For all ELF files the
// dynamically linked shared objects are collected and added to the libsDir
// directory. The paths to the directories they have been found at are added as
// symlinks to the libsDir directory as well.
func New(ctx context.Context, cfg Spec) (*FS, error) {
	libs, err := sys.CollectLibsFor(ctx, cfg.executables()...)
	if err != nil {
		return nil, fmt.Errorf("collect libs: %w", err)
	}

	fsys := virtfs.New()

	entries := fsEntries(cfg, libs)
	for _, entry := range entries {
		err := entry(fsys)
		if err != nil {
			return nil, fmt.Errorf("add to fs: %w", err)
		}
	}

	return fsys, nil
}

func fsEntries(cfg Spec, libs sys.LibCollection) []virtfs.EntryFunc {
	copyFile := func(name, src string) virtfs.EntryFunc {
		return virtfs.Copy(name, func() (fs.File, error) {
			return cfg.Fsys.Open(deroot(src))
		})
	}

	entries := []virtfs.EntryFunc{
		virtfs.MkdirAll(dataDir),
		virtfs.MkdirAll(confDir),
		virtfs.MkdirAll(libsDir),
		virtfs.MkdirAll(modulesDir),
		virtfs.MkdirAll("run"),
		virtfs.MkdirAll("tmp"),
		virtfs.Write(hostsPath, []byte(hostsFile)),
		virtfs.Write(configPath, cfg.Config),
		copyFile(mainPath, cfg.Executable),
	}

	if cfg.Init != nil {
		entries = append(entries, virtfs.Write(initPath, cfg.Init))
	} else {
		entries = append(entries, virtfs.Symlink(mainPath, initPath))
	}

	for _, path := range cfg.Files {
		entries = append(entries, copyFile(replaceDir(dataDir, path), path))
	}

	for idx, path := range cfg.Modules {
		name := fmt.Sprintf("%04d-%s", idx, filepath.Base(path))
		entries = append(entries, copyFile(replaceDir(modulesDir, name), path))
	}

	for path := range libs.Libs() {
		entries = append(entries, copyFile(replaceDir(libsDir, path), path))
	}

	for path := range libs.SearchPaths() {
		path = deroot(path)
		if path == libsDir {
			continue
		}

		entries = append(entries,
			virtfs.MkdirAll(filepath.Dir(path)),
			virtfs.MayExist(virtfs.Symlink(root(libsDir), path)),
		)
	}

	return entries
}

func root(s string) string {
	return filepath.Join(string(filepath.Separator), s)
}

func deroot(s string) string {
	return strings.TrimPrefix(s, string(filepath.Separator))
}

func replaceDir(dir, path string) string {
	return filepath.Join(dir, filepath.Base(path))
}
