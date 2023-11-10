package files

import (
	"debug/elf"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/exp/slices"
)

// TODO: Instead of trying to resolve shared objects ourself, which is quite
// fragile, use the actual shared object resolving by the actual interpreter.
// The elf.File has the actually used interpreter in the progs slice wit type
// elf.PT_INTERP. Call it with flag "--list" to get the actual resolved,
// deduplicated list of libs.

// ELFLibResolver resolves dynamically linked libraries of ELF file. It collects
// the libraries deduplicated for all files resolved with
// [ELFLibResolver.Resolve].
type ELFLibResolver struct {
	SearchPaths []string
	Libs        []string
}

// Resolve analyzes the required linked libraries of the ELF file with the
// given path. The libraries are search for in the library search paths and
// are added with their absolute path to [ELFLibResolver]'s list of libs. Call
// [ELFLibResolver.Libs] once all files are resolved.
func (r *ELFLibResolver) Resolve(elfFile string) error {
	libs, err := LinkedLibs(elfFile)
	if err != nil {
		return fmt.Errorf("get linked libs: %v", err)
	}

	for _, lib := range libs {
		var found bool
		for _, searchPath := range r.SearchPaths {
			path := filepath.Join(searchPath, lib)
			_, err := os.Stat(path)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					continue
				}
				return err
			}
			if !slices.Contains(r.Libs, path) {
				r.Libs = append(r.Libs, path)
				if err := r.Resolve(path); err != nil {
					return err
				}
			}
			found = true
			break
		}
		if !found {
			return fmt.Errorf("lib could not be resolved: %s", lib)
		}
	}

	return nil
}

// LinkedLibs fetches the list of dynamically linked libraries from the ELF
// file.
func LinkedLibs(elfFilePath string) ([]string, error) {
	elfFile, err := elf.Open(elfFilePath)
	if err != nil {
		return nil, err
	}
	defer elfFile.Close()

	libs, err := elfFile.ImportedLibraries()
	if err != nil {
		return nil, fmt.Errorf("read libs: %v", err)
	}

	return libs, nil
}
