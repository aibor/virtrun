// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sys

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"maps"
	"path/filepath"
	"slices"
)

// LibCollection is a deduplicated collection of dynamically linked libraries
// and paths they are found at.
type LibCollection struct {
	libs        map[string]int
	searchPaths map[string]int
}

// Libs returns an iterator that iterates all libraries sorted by path.
func (c *LibCollection) Libs() iter.Seq[string] {
	return func(yield func(string) bool) {
		for _, name := range slices.Sorted(maps.Keys(c.libs)) {
			if !yield(name) {
				return
			}
		}
	}
}

// SearchPaths returns an iterator that iterates all search paths sorted by
// path.
func (c *LibCollection) SearchPaths() iter.Seq[string] {
	return func(yield func(string) bool) {
		for _, name := range slices.Sorted(maps.Keys(c.searchPaths)) {
			if !yield(name) {
				return
			}
		}
	}
}

// CollectLibsFor recursively resolves the dynamically linked shared objects of
// all given ELF files.
//
// The dynamic linker consumed LD_LIBRARY_PATH from the environment.
func CollectLibsFor(
	ctx context.Context,
	files ...string,
) (LibCollection, error) {
	collection := LibCollection{
		libs:        make(map[string]int),
		searchPaths: make(map[string]int),
	}

	for _, name := range files {
		err := collectLibsFor(ctx, collection.libs, name)
		if err != nil {
			return collection, fmt.Errorf("[%s]: %w", name, err)
		}
	}

	for name := range collection.libs {
		dir, _ := filepath.Split(name)

		err := collectSearchPathsFor(collection.searchPaths, dir)
		if err != nil {
			return collection, fmt.Errorf("[%s]: %w", name, err)
		}
	}

	return collection, nil
}

func collectLibsFor(
	ctx context.Context,
	libs map[string]int,
	name string,
) error {
	// For each regular file, try to get linked shared objects.
	// Ignore if it is not an ELF file or if it is statically linked (has no
	// interpreter). Collect the absolute paths of the found shared objects
	// deduplicated in a set.
	paths, err := Ldd(ctx, name)
	if err != nil {
		if errors.Is(err, ErrNotELFFile) ||
			errors.Is(err, ErrNoInterpreter) {
			return nil
		}

		return err
	}

	for _, p := range paths {
		absPath, err := AbsolutePath(p)
		if err != nil {
			return err
		}

		libs[absPath]++
	}

	return nil
}

func collectSearchPathsFor(paths map[string]int, dir string) error {
	dir = filepath.Clean(dir)
	if dir == "" {
		return nil
	}

	paths[dir]++

	// Try if the directory has symbolic links and resolve them, so we
	// get the real path that the dynamic linker needs.
	canonicalDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return fmt.Errorf("resolve symlinks: %w", err)
	}

	if canonicalDir != dir {
		paths[canonicalDir]++
	}

	return nil
}
