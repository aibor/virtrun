package initramfs

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/aibor/virtrun/internal/files"
)

const (
	// LibsDir is the archive's directory for all dynamically linked libraries.
	LibsDir = "lib"
	// AdditionalFilesDir is the archive's directory for all additional files
	// beside the init file.
	FilesDir = "files"
)

// Archive represents a file tree that can be used as an initramfs for the
// Linux kernel.
//
// Create a new instance using [New]. Additional files can be added with
// [Archive.AddFiles]. Dynamically linked ELF libraries can be resolved and
// added for all already added files by calling [Archive.ResolveLinkedLibs].
// Once ready, write the [Archive] with [Archive.WriteCPIO].
type Archive struct {
	fileTree files.Tree
	sourceFS fs.FS
}

// New creates a new [Archive] with the given file added as "/init".
// The file path must be absolute or relative to "/".
func New(initFilePath string) *Archive {
	a := Archive{sourceFS: os.DirFS("/")}
	// This can never fail on a new tree.
	_, _ = a.fileTree.GetRoot().AddFile("init", initFilePath)
	return &a
}

// NewWithEmbedded creates a new [Archive] with the given [fs.File] used as
// "/init".
//
// The given file must be statically linked.
func NewWithEmbedded(init fs.File) *Archive {
	a := Archive{sourceFS: os.DirFS("/")}
	// This can never fail on a new tree.
	_, _ = a.fileTree.GetRoot().AddVirtualFile("init", init)
	return &a
}

// AddFile creates [FilesDir] and adds the given file to it. If name is empty
// the base name of the file is used.
// The file path must be absolute or relative to "/".
func (a *Archive) AddFile(name, path string) error {
	if name == "" {
		name = filepath.Base(path)
	}
	return a.withDirEntry(FilesDir, func(dirEntry *files.Entry) error {
		return addFile(dirEntry, name, path)
	})
}

// AddFiles creates [FilesDir] and adds the given files to it.
// The file paths must be absolute or relative to "/".
func (a *Archive) AddFiles(paths ...string) error {
	return a.withDirEntry(FilesDir, func(dirEntry *files.Entry) error {
		for _, file := range paths {
			if err := addFile(dirEntry, filepath.Base(file), file); err != nil {
				return err
			}
		}
		return nil
	})
}

// AddRequiredSharedObjects recursively resolves the dynamically linked
// shared objects of all ELF files in the [Archive].
//
// The dynamic linker consumed LD_LIBRARY_PATH from the environment.
// Resolved libraries are added to [LibsDir]. For each search path a symbolic
// link is added pointing to [LibsDir].
func (a *Archive) AddRequiredSharedObjects() error {
	// Walk file tree. For each regular file, try to get linked shared objects.
	// Ignore if it is not an ELF file or if it is statically linked (has no
	// interpreter). Collect the absolute paths of the found shared objects
	// deduplicated in a set.
	pathSet := make(map[string]bool)
	if err := a.fileTree.Walk(func(path string, entry *files.Entry) error {
		if entry.Type != files.TypeRegular {
			return nil
		}
		paths, err := files.Ldd(entry.RelatedPath)
		if err != nil {
			if err == files.ErrNotELFFile || err == files.ErrNoInterpreter {
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
		return err
	}

	absLibDir := filepath.Join(string(filepath.Separator), LibsDir)
	addLinkToLibDir := func(dir string) error {
		if dir == "" || dir == absLibDir {
			return nil
		}
		err := a.fileTree.Ln(absLibDir, dir)
		if err != nil && err != files.ErrEntryExists {
			return fmt.Errorf("add link for %s: %v", dir, err)
		}
		return nil
	}

	// Walk the found shared object paths and add all to the central lib dir.
	// In order to keep any references and search paths of the dynamic linker
	// working, add symbolic links for all other directories where libs are
	// copied from to the central lib dir.
	if err := a.withDirEntry(LibsDir, func(dirEntry *files.Entry) error {
		for path := range pathSet {
			dir, name := filepath.Split(path)
			if _, err := dirEntry.AddFile(name, path); err != nil {
				return fmt.Errorf("add file %s: %v", name, err)
			}
			if err := addLinkToLibDir(dir); err != nil {
				return err
			}
			// Try if the directory has symbolic links and resolve them, so we
			// get the real path that the dynamic linker needs.
			canonicalDir, err := filepath.EvalSymlinks(dir)
			if err != nil {
				return err
			}
			if err := addLinkToLibDir(canonicalDir); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

// WriteCPIO writes the [Archive] as CPIO archive to the given writer.
func (a *Archive) WriteCPIO(writer io.Writer) error {
	w := NewCPIOWriter(writer)
	defer w.Close()
	return a.writeTo(w)
}

func (a *Archive) writeTo(writer Writer) error {
	return a.fileTree.Walk(func(path string, entry *files.Entry) error {
		switch entry.Type {
		case files.TypeRegular:
			// Cut leading / since fs.FS considers it invalid.
			relPath := strings.TrimPrefix(entry.RelatedPath, "/")
			source, err := a.sourceFS.Open(relPath)
			if err != nil {
				return err
			}
			defer source.Close()
			return writer.WriteRegular(path, source, 0755)
		case files.TypeDirectory:
			return writer.WriteDirectory(path)
		case files.TypeLink:
			return writer.WriteLink(path, entry.RelatedPath)
		case files.TypeVirtual:
			return writer.WriteRegular(path, entry.Source, 0755)
		default:
			return fmt.Errorf("unknown file type %d", entry.Type)
		}
	})
}

func (a *Archive) withDirEntry(dir string, fn func(*files.Entry) error) error {
	dirEntry, err := a.fileTree.Mkdir(dir)
	if err != nil {
		return fmt.Errorf("add dir %s: %v", dir, err)
	}
	return fn(dirEntry)
}

func addFile(dirEntry *files.Entry, name, path string) error {
	if _, err := dirEntry.AddFile(name, path); err != nil {
		return fmt.Errorf("add file %s: %v", path, err)
	}
	return nil
}
