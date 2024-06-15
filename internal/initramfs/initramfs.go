// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package initramfs

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const fileMode = 0o755

// Initramfs represents a file tree that can be used as an initramfs for the
// Linux kernel.
//
// Create a new instance using [New]. Additional files can be added with
// [Initramfs.AddFiles]. Dynamically linked ELF libraries can be resolved
// and added for all already added regular files by calling
// [Initramfs.AddRequiredSharedObjects]. Once ready, write the [Initramfs] with
// [Initramfs.WriteInto].
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

	return i.withDirNode(dir, func(dirNode *TreeNode) error {
		return addFile(dirNode, name, path)
	})
}

// AddFiles creates [Initramfs.filesDir] and adds the given files to it.
// The file paths must be absolute or relative to "/".
func (i *Initramfs) AddFiles(dir string, paths ...string) error {
	return i.withDirNode(dir, func(dirNode *TreeNode) error {
		for _, file := range paths {
			if err := addFile(dirNode, filepath.Base(file), file); err != nil {
				return err
			}
		}

		return nil
	})
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
	if err := i.withDirNode(i.libDir, func(dirNode *TreeNode) error {
		for path := range pathSet {
			dir, name := filepath.Split(path)
			if _, err := dirNode.AddRegular(name, path); err != nil {
				return fmt.Errorf("add file %s: %v", name, err)
			}
			if err := i.addLinkToLibDir(dir); err != nil {
				return err
			}
			// Try if the directory has symbolic links and resolve them, so we
			// get the real path that the dynamic linker needs.
			canonicalDir, err := filepath.EvalSymlinks(dir)
			if err != nil {
				return err
			}
			if err := i.addLinkToLibDir(canonicalDir); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return err
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
	if err := i.fileTree.Walk(func(path string, node *TreeNode) error {
		if node.Type != FileTypeRegular {
			return nil
		}

		paths, err := Ldd(node.RelatedPath)
		if err != nil {
			if errors.Is(err, ErrNotELFFile) || errors.Is(err, ErrNoInterpreter) {
				return nil
			}

			return fmt.Errorf("resolve %s: %v", path, err)
		}

		for _, p := range paths {
			absPath, err := filepath.Abs(p)
			if err != nil {
				return fmt.Errorf("abs path for %s: %v", p, err)
			}
			pathSet[absPath] = true
		}

		return nil
	}); err != nil {
		return nil, err
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
	if err != nil && !errors.Is(err, ErrNodeExists) {
		return fmt.Errorf("add link for %s: %v", path, err)
	}

	return nil
}

// WriteToTempFile writes the complete CPIO archive into a new file in the
// given directory and returns its filename. If tmpDir is the empty string the
// default directory is used as returned by [os.TempDir].
// The caller is responsible for removing the file once it is not needed
// anymore.
func (i *Initramfs) WriteToTempFile(tmpDir string) (string, error) {
	file, err := os.CreateTemp(tmpDir, "initramfs")
	if err != nil {
		return "", fmt.Errorf("create temp file: %v", err)
	}
	defer file.Close()

	err = i.WriteInto(file)
	if err != nil {
		_ = os.Remove(file.Name())

		return "", fmt.Errorf("create archive: %v", err)
	}

	return file.Name(), nil
}

// WriteInto writes the [Initramfs] as CPIO archive to the given writer.
func (i *Initramfs) WriteInto(writer io.Writer) error {
	w := NewCPIOWriter(writer)
	defer w.Close()

	return i.writeTo(w, os.DirFS("/"))
}

// writeTo writes all collected files into the given writer. Regular files are
// copied from the given sourceFS.
func (i *Initramfs) writeTo(writer Writer, sourceFS fs.FS) error {
	return i.fileTree.Walk(func(path string, node *TreeNode) error {
		switch node.Type {
		case FileTypeRegular:
			// Cut leading / since fs.FS considers it invalid.
			relPath := strings.TrimPrefix(node.RelatedPath, "/")

			source, err := sourceFS.Open(relPath)
			if err != nil {
				return err
			}
			defer source.Close()

			return writer.WriteRegular(path, source, fileMode)
		case FileTypeDirectory:
			return writer.WriteDirectory(path)
		case FileTypeLink:
			return writer.WriteLink(path, node.RelatedPath)
		case FileTypeVirtual:
			return writer.WriteRegular(path, node.Source, fileMode)
		default:
			return fmt.Errorf("unknown file type %d", node.Type)
		}
	})
}

func (i *Initramfs) withDirNode(dir string, fn func(*TreeNode) error) error {
	dirNode, err := i.fileTree.Mkdir(dir)
	if err != nil {
		return fmt.Errorf("add dir %s: %v", dir, err)
	}

	return fn(dirNode)
}

func addFile(dirNode *TreeNode, name, path string) error {
	if _, err := dirNode.AddRegular(name, path); err != nil {
		return fmt.Errorf("add file %s: %v", path, err)
	}

	return nil
}
