package files

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"golang.org/x/exp/maps"
)

// Entry is a single file tree entry.
type Entry struct {
	// Type of this entry.
	Type Type
	// Related path depending on the file type. Empty for directories,
	// target path for links, source files for regular files.
	RelatedPath string
	// Source is the content for a virtual regular file.
	Source fs.File

	children map[string]*Entry
}

// String returns a string representation of the Entry.
func (e *Entry) String() string {
	switch e.Type {
	case TypeRegular:
		return "File from: " + e.RelatedPath
	case TypeDirectory:
		return fmt.Sprintf("Dir with entries: % s", maps.Keys(e.children))
	case TypeLink:
		return "Link to: " + e.RelatedPath
	case TypeVirtual:
		return "File virtual"
	default:
		return "invalid type"
	}
}

// IsDir returns true if the [Entry] is a directory.
func (e *Entry) IsDir() bool {
	return e.Type == TypeDirectory
}

// IsLink returns true if the [Entry] is a link.
func (e *Entry) IsLink() bool {
	return e.Type == TypeLink
}

// IsRegular returns true if the [Entry] is a regular file.
func (e *Entry) IsRegular() bool {
	return e.Type == TypeRegular
}

// IsVirtual returns true if the [Entry] is a virtual regular file.
func (e *Entry) IsVirtual() bool {
	return e.Type == TypeVirtual
}

// AddFile adds a new regular file [Entry] children.
func (e *Entry) AddFile(name, relatedPath string) (*Entry, error) {
	entry := &Entry{
		Type:        TypeRegular,
		RelatedPath: relatedPath,
	}
	return e.AddEntry(name, entry)
}

// AddDirectory adds a new directory [Entry] children.
func (e *Entry) AddDirectory(name string) (*Entry, error) {
	entry := &Entry{
		Type: TypeDirectory,
	}
	return e.AddEntry(name, entry)
}

// AddLink adds a new link [Entry] children.
func (e *Entry) AddLink(name, relatedPath string) (*Entry, error) {
	entry := &Entry{
		Type:        TypeLink,
		RelatedPath: relatedPath,
	}
	return e.AddEntry(name, entry)
}

// AddVirtualFile adds a new virtual regular file [Entry] children.
func (e *Entry) AddVirtualFile(name string, source fs.File) (*Entry, error) {
	entry := &Entry{
		Type:   TypeVirtual,
		Source: source,
	}
	return e.AddEntry(name, entry)
}

// AddEntry adds an arbitrary [Entry] as children. The caller is responsible
// for using only valid [Type]s and according fields.
func (e *Entry) AddEntry(name string, entry *Entry) (*Entry, error) {
	if !e.IsDir() {
		return nil, ErrEntryNotDir
	}
	if ee, exists := e.children[name]; exists {
		return ee, ErrEntryExists
	}
	if e.children == nil {
		e.children = make(map[string]*Entry)
	}
	e.children[name] = entry
	return entry, nil
}

// GetEntry getsan [Entry] for the given name. Return ErrEntryNotExists if it
// doesn't exist.
func (e *Entry) GetEntry(name string) (*Entry, error) {
	if !e.IsDir() {
		return nil, ErrEntryNotDir
	}
	entry, exists := e.children[name]
	if !exists {
		return nil, ErrEntryNotExists
	}
	return entry, nil
}

func (e *Entry) walk(base string, fn WalkFunc) error {
	for name, entry := range e.children {
		path := filepath.Join(base, name)
		if err := fn(path, entry); err != nil {
			return err
		}
		if entry.IsDir() {
			if err := entry.walk(path, fn); err != nil {
				return err
			}
		}
	}
	return nil
}
