package internal

import (
	"debug/elf"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var LibSearchPaths string = "/usr/lib:/usr/lib64:/lib:/lib64"

// ResolveLinkedLibs resolves dynamic libraries recursively.
//
// It returns a list of absolute paths to the linked libraries.
func ResolveLinkedLibs(fileName string) ([]string, error) {
	elfFile, err := elf.Open(fileName)
	if err != nil {
		return nil, err
	}

	libs, err := elfFile.ImportedLibraries()
	if err != nil {
		return nil, fmt.Errorf("read libs: %v", err)
	}

	searchPaths := strings.Split(LibSearchPaths, ":")
	libPaths := make(map[string]bool, 0)
	for _, lib := range libs {
		for _, searchPath := range searchPaths {
			path := filepath.Join(searchPath, lib)
			lp, err := ResolveLinkedLibs(path)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					continue
				}
				return nil, err
			}
			libPaths[path] = true
			for _, p := range lp {
				libPaths[p] = true
			}
			break
		}
	}

	l := make([]string, 0)
	for p := range libPaths {
		l = append(l, p)
	}
	return l, nil
}
