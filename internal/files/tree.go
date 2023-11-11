package files

import (
	"fmt"
	"path/filepath"
)

// Tree represents a simple file tree.
type Tree struct {
	// Do not access directly! Always use [Tree.GetRoot] to access the root
	// entry to ensure it exists.
	root *Entry
}

func isRoot(path string) bool {
	switch filepath.Clean(path) {
	case "", ".", string(filepath.Separator):
		return true
	default:
		return false
	}
}

// GetRoot returns the root entry of the tree.
func (t *Tree) GetRoot() *Entry {
	if t.root == nil {
		t.root = &Entry{
			Type: TypeDirectory,
		}
	}
	return t.root
}

// GetEntry returns the entry for the given path. Returns ErrEntryNotExists if
// the entry does not exist.
func (t *Tree) GetEntry(path string) (*Entry, error) {
	if isRoot(path) {
		return t.GetRoot(), nil
	}
	dir, name := filepath.Split(filepath.Clean(path))
	parent, err := t.GetEntry(dir)
	if err != nil {
		return nil, err
	}
	return parent.GetEntry(name)
}

// Mkdir adds a directory entry for the given path. Non existing parents
// are created recursively. If any of the parents exists but is not a directory
// ErrEntryNotDir is returned.
func (t *Tree) Mkdir(path string) (*Entry, error) {
	cleaned := filepath.Clean(path)
	if isRoot(cleaned) {
		return t.GetRoot(), nil
	}
	dir, name := filepath.Split(cleaned)
	parent, err := t.Mkdir(dir)
	if err != nil {
		return nil, fmt.Errorf("mkdir %s: %v", dir, err)
	}
	entry, err := parent.AddDirectory(name)
	if err == ErrEntryExists && entry.IsDir() {
		err = nil
	}
	return entry, err
}

// Ln adds links to target for the given path.
func (t *Tree) Ln(target string, path string) error {
	cleaned := filepath.Clean(path)
	dir, name := filepath.Split(cleaned)
	dirEntry, err := t.Mkdir(dir)
	if err != nil {
		return err
	}
	if l, err := dirEntry.AddLink(name, target); err != nil {
		if err != ErrEntryExists || !l.IsLink() {
			return err
		}
	}
	return nil
}

// WalkFunc is called with the absolute path to the entry.
type WalkFunc func(path string, entry *Entry) error

// Walk walks the tree recursively, starting at the root, and runs the given
// function for each entry. If the function returns an error, the recursion is
// terminated immediately and the error is returned.
func (f *Tree) Walk(fn WalkFunc) error {
	return f.GetRoot().walk(string(filepath.Separator), fn)
}
