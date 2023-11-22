package initramfs

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/aibor/virtrun/internal/archive"
	"github.com/aibor/virtrun/internal/files"
)

// InitramfsOption is a functional option that can be passed to the constructor.
type InitramfsOption func(*Initramfs)

// WithLibsDir sets the directory for all collected shared libraries.
func WithLibsDir(dir string) InitramfsOption {
	return func(i *Initramfs) {
		i.libsDir = dir
	}
}

// WithFilesDir sets the directory where all additional files are placed into.
func WithFilesDir(dir string) InitramfsOption {
	return func(i *Initramfs) {
		i.filesDir = dir
	}
}

// InitFile defines how the file used as init program at "/init" is created.
type InitFile func(*files.Entry) (*files.Entry, error)

// InitFilePath creates the "/init" file copied from a real path. With this
// required shared libraries can be resolved and added to the Initramfs.
func InitFilePath(path string) InitFile {
	return func(rootDir *files.Entry) (*files.Entry, error) {
		return rootDir.AddFile("init", path)
	}
}

// InitFileVirtual creates the "/init" file from an [fs.File]. This must be
// a statically linked binary or it will not start correctly unless the required
// shared libraries are added manually.
func InitFileVirtual(file fs.File) InitFile {
	return func(rootDir *files.Entry) (*files.Entry, error) {
		return rootDir.AddVirtualFile("init", file)
	}
}

// Initramfs represents a file tree that can be used as an initramfs for the
// Linux kernel.
//
// Create a new instance using [New]. Additional files can be added with
// [Initramfs.AddFiles]. Dynamically linked ELF libraries can be resolved
// and added for all already added regular files by calling
// [Initramfs.AddRequiredSharedObjects]. Once ready, write the [Initramfs] with
// [Initramfs.WriteInto].
type Initramfs struct {
	// LibsDir is the archive's directory for all dynamically linked libraries.
	libsDir string
	// AdditionalFilesDir is the archive's directory for all additional files
	// beside the init file.
	filesDir string
	fileTree files.Tree
}

// New creates a new [Initramfs].
//
// The init file is created from the given [InitFile] function.
func New(initFile InitFile, opts ...InitramfsOption) *Initramfs {
	i := &Initramfs{
		libsDir:  "lib",
		filesDir: "files",
	}
	for _, opt := range opts {
		opt(i)
	}

	// This can never fail on a new tree.
	_, _ = initFile(i.fileTree.GetRoot())
	return i
}

// AddFile creates [Initramfs.filesDir] and adds the given file to it. If name
// is empty the base name of the file is used.
// The file path must be absolute or relative to "/".
func (i *Initramfs) AddFile(name, path string) error {
	if name == "" {
		name = filepath.Base(path)
	}
	return i.withDirEntry(i.filesDir, func(dirEntry *files.Entry) error {
		return addFile(dirEntry, name, path)
	})
}

// AddFiles creates [Initramfs.filesDir] and adds the given files to it.
// The file paths must be absolute or relative to "/".
func (i *Initramfs) AddFiles(paths ...string) error {
	return i.withDirEntry(i.filesDir, func(dirEntry *files.Entry) error {
		for _, file := range paths {
			if err := addFile(dirEntry, filepath.Base(file), file); err != nil {
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
	// Walk file tree. For each regular file, try to get linked shared objects.
	// Ignore if it is not an ELF file or if it is statically linked (has no
	// interpreter). Collect the absolute paths of the found shared objects
	// deduplicated in a set.
	pathSet := make(map[string]bool)
	if err := i.fileTree.Walk(func(path string, entry *files.Entry) error {
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

	absLibDir := filepath.Join(string(filepath.Separator), i.libsDir)
	addLinkToLibDir := func(dir string) error {
		if dir == "" || dir == absLibDir {
			return nil
		}
		err := i.fileTree.Ln(absLibDir, dir)
		if err != nil && err != files.ErrEntryExists {
			return fmt.Errorf("add link for %s: %v", dir, err)
		}
		return nil
	}

	// Walk the found shared object paths and add all to the central lib dir.
	// In order to keep any references and search paths of the dynamic linker
	// working, add symbolic links for all other directories where libs are
	// copied from to the central lib dir.
	if err := i.withDirEntry(i.libsDir, func(dirEntry *files.Entry) error {
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

// WriteInto writes the [Initramfs] as CPIO archive to the given writer.
func (i *Initramfs) WriteInto(writer io.Writer) error {
	w := archive.NewCPIOWriter(writer)
	defer w.Close()
	return i.writeTo(w, os.DirFS("/"))
}

// writeTo writes all collected files into the given writer. Regular files are
// copied from the given sourceFS.
func (i *Initramfs) writeTo(writer archive.Writer, sourceFS fs.FS) error {
	return i.fileTree.Walk(func(path string, entry *files.Entry) error {
		switch entry.Type {
		case files.TypeRegular:
			// Cut leading / since fs.FS considers it invalid.
			relPath := strings.TrimPrefix(entry.RelatedPath, "/")
			source, err := sourceFS.Open(relPath)
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

func (i *Initramfs) withDirEntry(dir string, fn func(*files.Entry) error) error {
	dirEntry, err := i.fileTree.Mkdir(dir)
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
