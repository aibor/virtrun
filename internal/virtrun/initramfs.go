// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package virtrun

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/aibor/virtrun/internal/sys"
	"github.com/aibor/virtrun/internal/virtfs"
)

const (
	dataDir       = "data"
	libsDir       = "lib"
	modulesDir    = "lib/modules"
	initPath      = "init"
	mainPath      = "main"
	archivePrefix = "virtrun-initramfs"
)

// Initramfs specifies the input for initramfs archive creation.
type Initramfs struct {
	// Binary is the main binary that is either called directly or by the init
	// program depending on the StandaloneInit flag.
	Binary string

	// Files is a list of any additional files that should be added to the
	// dataDir directory. For ELF files the required dynamic libraries are
	// added the libsDir directory.
	Files []string

	// Modules is a list of kernel module files. They are added to the
	// modulesDir directory.
	Modules []string

	// Fsys is the file system all files should be copied from.
	Fsys fs.FS

	// StandaloneInit determines if the main Binary should be called as init
	// directly. The main binary is responsible for a clean shutdown of the
	// system.
	StandaloneInit bool

	// Keep determines if the archive file is removed by the cleanup function
	// returned by [BuildInitramfsArchive]. If set to true, the file is not
	// removed. Instead, a log message with the file's path is printed.
	Keep bool
}

func (i Initramfs) binaries() []string {
	binaryFiles := make([]string, 0, 1+len(i.Files))
	binaryFiles = append(binaryFiles, i.Binary)
	binaryFiles = append(binaryFiles, i.Files...)

	return binaryFiles
}

// BuildInitramfsArchive creates a new initramfs CPIO archive file.
//
// The archive consists of a main binary that is either called directly or
// by the init program. All other files are added to the dataDir directory.
// Kernel modules are added to modulesDir directory. For all ELF files the
// dynamically linked shared objects are collected and added to the libsDir
// directory. The paths to the directories they have been found at are added as
// symlinks to the libsDir directory as well.
//
// The CPIO archive is written to [os.TempDir]. The path to the file is
// returned along with a cleanup function. The caller is responsible to call
// the function once the archive file is no longer needed.
func BuildInitramfsArchive(
	ctx context.Context,
	cfg Initramfs,
	initProg fs.File,
) (string, func() error, error) {
	libs, err := sys.CollectLibsFor(ctx, cfg.binaries()...)
	if err != nil {
		return "", nil, fmt.Errorf("collect libs: %w", err)
	}

	fsys := virtfs.New()

	entries := fsEntries(cfg, libs, initProg)
	for _, entry := range entries {
		err := entry.addTo(fsys)
		if err != nil {
			return "", nil, fmt.Errorf("add to fs: %w", err)
		}
	}

	path, err := WriteToTempFile(fsys, "", archivePrefix)
	if err != nil {
		return "", nil, err
	}

	slog.Debug("Created initramfs archive", slog.String("path", path))

	var removeFn func() error

	if cfg.Keep {
		removeFn = func() error {
			slog.Info("Keep initramfs archive", slog.String("path", path))
			return nil
		}
	} else {
		removeFn = func() error {
			slog.Debug("Remove initramfs archive", slog.String("path", path))
			return os.Remove(path)
		}
	}

	return path, removeFn, nil
}

func fsEntries(
	cfg Initramfs,
	libs sys.LibCollection,
	initProg fs.File,
) []fsEntry {
	entries := []fsEntry{
		directory(dataDir),
		directory(libsDir),
		directory(modulesDir),
		directory("run"),
		directory("tmp"),
	}

	binaryPath := initPath
	if initProg != nil {
		binaryPath = mainPath

		entries = append(entries, file{
			Path: initPath,
			OpenFn: func() (fs.File, error) {
				return initProg, nil
			},
		})
	}

	entries = append(entries, copyFile{
		Source: deroot(cfg.Binary),
		Dest:   binaryPath,
		Fsys:   cfg.Fsys,
	})

	for _, path := range cfg.Files {
		entries = append(entries, copyFile{
			Source: deroot(path),
			Dest:   replaceDir(dataDir, path),
			Fsys:   cfg.Fsys,
		})
	}

	for idx, path := range cfg.Modules {
		name := fmt.Sprintf("%04d-%s", idx, filepath.Base(path))
		entries = append(entries, copyFile{
			Source: deroot(path),
			Dest:   replaceDir(modulesDir, name),
			Fsys:   cfg.Fsys,
		})
	}

	for path := range libs.Libs() {
		entries = append(entries, copyFile{
			Source: deroot(path),
			Dest:   replaceDir(libsDir, path),
			Fsys:   cfg.Fsys,
		})
	}

	for path := range libs.SearchPaths() {
		path = deroot(path)
		if path == libsDir {
			continue
		}

		entries = append(entries,
			directory(filepath.Dir(path)),
			symlink{
				Target:   root(libsDir),
				Path:     path,
				MayExist: true,
			},
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
