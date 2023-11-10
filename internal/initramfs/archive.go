package initramfs

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/exp/slices"

	"github.com/aibor/virtrun/internal/files"
)

const (
	// LibsDir is the archive's directory for all dynamically linked libraries.
	LibsDir = "lib"
	// AdditionalFilesDir is the archive's directory for all additional files
	// beside the init file.
	FilesDir = "files"
	// LibSearchPath defines the directories to lookup linked libraries.
	LibSearchPath = "/lib64:/lib/x86_64-linux-gnu:/usr/lib64:/lib:/usr/lib"
)

// Write resolves ELF dynamically linked libraries of all currently added files
// and writes the initramfs to a file in [os.TempDir]. The optional
// libsearchpath is a colon separated string that specifies the directories to
// search libraries in. Same format as LD_LIBRARY_PATH has. It is the caller's
// responsibility to remove the file when it is no longer needed.
func (a *Archive) Write(libsearchpath string) (string, error) {
	var err error
	if err := a.ResolveLinkedLibs(libsearchpath); err != nil {
		return "", fmt.Errorf("resolve: %v", err)
	}

	archiveFile, err := os.CreateTemp("", "initramfs")
	if err != nil {
		return "", fmt.Errorf("create file")
	}
	defer archiveFile.Close()

	if err := a.WriteCPIO(archiveFile); err != nil {
		_ = os.Remove(archiveFile.Name())
		return "", fmt.Errorf("write: %v", err)
	}

	return archiveFile.Name(), nil
}

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

// ResolveLinkedLibs recursively resolves the dynamically linked libraries of
// all regular files in the [Archive].
//
// If the given searchPath string is empty the default [LibSearchPath] is used.
// Resolved libraries are added to [LibsDir]. For each search path a symoblic
// link is added pointiong to [LibsDir].
func (a *Archive) ResolveLinkedLibs(searchPath string) error {
	if searchPath == "" {
		searchPath = LibSearchPath
	}
	searchPaths := filepath.SplitList(searchPath)
	searchPaths = slices.DeleteFunc(searchPaths, func(e string) bool { return e == "" })

	resolver := files.ELFLibResolver{
		SearchPaths: searchPaths,
	}

	err := a.fileTree.Walk(func(path string, entry *files.Entry) error {
		if entry.Type != files.TypeRegular {
			return nil
		}
		return resolver.Resolve(entry.RelatedPath)
	})
	if err != nil {
		return fmt.Errorf("resolve: %v", err)
	}

	if err := a.withDirEntry(LibsDir, func(dirEntry *files.Entry) error {
		for _, lib := range resolver.Libs {
			name := filepath.Base(lib)
			if _, err := dirEntry.AddFile(name, lib); err != nil {
				return fmt.Errorf("add lib %s: %v", name, err)
			}
		}
		return nil
	}); err != nil {
		return err
	}

	absLibDir := filepath.Join(string(filepath.Separator), LibsDir)
	for _, searchPath := range searchPaths {
		err := a.fileTree.Ln(absLibDir, searchPath)
		if err != nil && err != files.ErrEntryExists {
			return fmt.Errorf("add link %s: %v", searchPath, err)
		}
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
