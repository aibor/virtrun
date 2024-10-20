// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package initramfs

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

const fileMode = 0o755

// Initramfs represents a file tree that can be used as an initramfs for the
// Linux kernel.
//
// Create a new instance using [New]. Additional files can be added with
// [Initramfs.AddFiles]. Dynamically linked ELF libraries can be resolved
// and added for all already added regular files by calling
// [Initramfs.AddRequiredSharedObjects]. Once ready, write the [Initramfs] with
// [Initramfs.WriteCPIOInto].
type Initramfs struct {
	fileTree Tree
	libDir   string
}

func WithRealInitFile(path string) func(*TreeNode) {
	return func(rootDir *TreeNode) {
		// Never fails on a new tree.
		_, _ = rootDir.AddRegular("init", path)
	}
}

func WithVirtualInitFile(file fs.File) func(*TreeNode) {
	return func(rootDir *TreeNode) {
		// Never fails on a new tree.
		_, _ = rootDir.AddVirtual("init", file)
	}
}

// New creates a new [Initramfs] with "/init" copied from the given file path.
func New(fn func(*TreeNode)) *Initramfs {
	initramfs := &Initramfs{
		libDir: filepath.Join(string(filepath.Separator), "lib"),
	}
	rootDir := initramfs.fileTree.GetRoot()

	fn(rootDir)

	return initramfs
}

// AddFile creates [Initramfs.filesDir] and adds the given file to it. If name
// is empty the base name of the file is used.
// The file path must be absolute or relative to "/".
func (i *Initramfs) AddFile(dir, name, path string) error {
	if name == "" {
		name = filepath.Base(path)
	}

	dirNode, err := i.mkdir(dir)
	if err != nil {
		return err
	}

	return addFile(dirNode, name, path)
}

// AddFiles creates [Initramfs.filesDir] and adds the given files to it.
// The file paths must be absolute or relative to "/".
func (i *Initramfs) AddFiles(dir string, paths ...string) error {
	dirNode, err := i.mkdir(dir)
	if err != nil {
		return err
	}

	for _, file := range paths {
		err := addFile(dirNode, filepath.Base(file), file)
		if err != nil {
			return err
		}
	}

	return nil
}

// AddRequiredSharedObjects recursively resolves the dynamically linked
// shared objects of all ELF files in the [Initramfs].
//
// The dynamic linker consumed LD_LIBRARY_PATH from the environment.
// Resolved libraries are added to [Initramfs.libsDir]. For each search path a
// symbolic link is added pointing to [Initramfs.libsDir].
func (i *Initramfs) AddRequiredSharedObjects() error {
	pathSet, err := i.collectLibs()
	if err != nil {
		return err
	}

	// Walk the found shared object paths and add all to the central lib dir.
	// In order to keep any references and search paths of the dynamic linker
	// working, add symbolic links for all other directories where libs are
	// copied from to the central lib dir.
	dirNode, err := i.mkdir(i.libDir)
	if err != nil {
		return err
	}

	for path := range pathSet {
		dir, name := filepath.Split(path)

		err := addFile(dirNode, name, path)
		if err != nil {
			return err
		}

		err = i.addLinkToLibDir(dir)
		if err != nil {
			return err
		}

		// Try if the directory has symbolic links and resolve them, so we
		// get the real path that the dynamic linker needs.
		canonicalDir, err := filepath.EvalSymlinks(dir)
		if err != nil {
			return &PathError{
				Op:   "eval symlinks",
				Path: path,
				Err:  err,
			}
		}

		err = i.addLinkToLibDir(canonicalDir)
		if err != nil {
			return err
		}
	}

	return nil
}

// collectLibs collects the libraries used by executables.
func (i *Initramfs) collectLibs() (map[string]bool, error) {
	pathSet := make(map[string]bool)

	// For each regular file, try to get linked shared objects.
	// Ignore if it is not an ELF file or if it is statically linked (has no
	// interpreter). Collect the absolute paths of the found shared objects
	// deduplicated in a set.
	for path, node := range i.fileTree.All() {
		if node.Type != TreeNodeTypeRegular {
			continue
		}

		paths, err := Ldd(node.RelatedPath)
		if err != nil {
			if errors.Is(err, ErrNotELFFile) ||
				errors.Is(err, ErrNoInterpreter) {
				continue
			}

			return nil, &PathError{
				Op:   "ldd",
				Path: path,
				Err:  err,
			}
		}

		for _, p := range paths {
			absPath, err := filepath.Abs(p)
			if err != nil {
				return nil, &PathError{
					Op:   "absolute",
					Path: path,
					Err:  err,
				}
			}

			pathSet[absPath] = true
		}
	}

	return pathSet, nil
}

// addLinkToLibDir adds a symbolic link for path to the central libraries
// directory.
func (i *Initramfs) addLinkToLibDir(path string) error {
	if path == "" || path == i.libDir {
		return nil
	}

	err := i.fileTree.Ln(i.libDir, path)
	if err != nil && !errors.Is(err, ErrTreeNodeExists) {
		return &PathError{
			Op:   "add link",
			Path: path,
			Err:  err,
		}
	}

	return nil
}

func (i *Initramfs) mkdir(dir string) (*TreeNode, error) {
	dirNode, err := i.fileTree.Mkdir(dir)
	if err != nil {
		return nil, &PathError{
			Op:   "mkdir",
			Path: dir,
			Err:  err,
		}
	}

	return dirNode, nil
}

// WriteToTempFile writes the complete CPIO archive into a new file in the
// given directory and returns its filename. If tmpDir is the empty string the
// default directory is used as returned by [os.TempDir].
// The caller is responsible for removing the file once it is not needed
// anymore.
func (i *Initramfs) WriteToTempFile(tmpDir string) (string, error) {
	file, err := os.CreateTemp(tmpDir, "initramfs")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer file.Close()

	err = i.WriteCPIOInto(file, os.DirFS("/"))
	if err != nil {
		_ = os.Remove(file.Name())
		return "", fmt.Errorf("create archive: %w", err)
	}

	return file.Name(), nil
}

// WriteCPIOInto writes the [Initramfs] as CPIO archive to the given [Writer]
// from the given source [fs.FS].
func (i *Initramfs) WriteCPIOInto(writer io.Writer, source fs.FS) error {
	w := NewCPIOWriter(writer)
	defer w.Close()

	return i.writeTo(w, source)
}

// writeTo writes all collected files into the given writer. Regular files are
// copied from the given source [fs.FS].
func (i *Initramfs) writeTo(writer Writer, source fs.FS) error {
	for path, node := range i.fileTree.All() {
		err := node.WriteTo(writer, path, source)
		if err != nil {
			return &PathError{
				Op:   "archive write",
				Path: path,
				Err:  err,
			}
		}
	}

	return nil
}

func addFile(dirNode *TreeNode, name, path string) error {
	_, err := dirNode.AddRegular(name, path)
	if err != nil {
		return &PathError{
			Op:   "add file",
			Path: path,
			Err:  err,
		}
	}

	return nil
}
