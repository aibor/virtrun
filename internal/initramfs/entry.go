// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package initramfs

import (
	"fmt"
	"io/fs"
	"path/filepath"
)

// TreeNode is a single file tree node.
type TreeNode struct {
	// Type of this node.
	Type FileType
	// Related path depending on the file type. Empty for directories,
	// target path for links, source files for regular files.
	RelatedPath string
	// Source is the content for a virtual regular file.
	Source fs.File

	children map[string]*TreeNode
}

// String returns a string representation of the TreeNode.
func (e *TreeNode) String() string {
	switch e.Type {
	case FileTypeRegular:
		return "File from: " + e.RelatedPath
	case FileTypeDirectory:
		keys := make([]string, 0, len(e.children))
		for key := range e.children {
			keys = append(keys, key)
		}

		return fmt.Sprintf("Dir with entries: % s", keys)
	case FileTypeLink:
		return "Link to: " + e.RelatedPath
	case FileTypeVirtual:
		return "File virtual"
	default:
		return "invalid type"
	}
}

// IsDir returns true if the [TreeNode] is a directory.
func (e *TreeNode) IsDir() bool {
	return e.Type == FileTypeDirectory
}

// IsLink returns true if the [TreeNode] is a link.
func (e *TreeNode) IsLink() bool {
	return e.Type == FileTypeLink
}

// IsRegular returns true if the [TreeNode] is a regular file.
func (e *TreeNode) IsRegular() bool {
	return e.Type == FileTypeRegular
}

// IsVirtual returns true if the [TreeNode] is a virtual regular file.
func (e *TreeNode) IsVirtual() bool {
	return e.Type == FileTypeVirtual
}

// AddRegular adds a new regular file [TreeNode] children.
func (e *TreeNode) AddRegular(name, relatedPath string) (*TreeNode, error) {
	node := &TreeNode{
		Type:        FileTypeRegular,
		RelatedPath: relatedPath,
	}

	return e.AddNode(name, node)
}

// AddDirectory adds a new directory [TreeNode] children.
func (e *TreeNode) AddDirectory(name string) (*TreeNode, error) {
	node := &TreeNode{
		Type: FileTypeDirectory,
	}

	return e.AddNode(name, node)
}

// AddLink adds a new link [TreeNode] children.
func (e *TreeNode) AddLink(name, relatedPath string) (*TreeNode, error) {
	node := &TreeNode{
		Type:        FileTypeLink,
		RelatedPath: relatedPath,
	}

	return e.AddNode(name, node)
}

// AddVirtual adds a new virtual file [TreeNode] children.
func (e *TreeNode) AddVirtual(name string, source fs.File) (*TreeNode, error) {
	node := &TreeNode{
		Type:   FileTypeVirtual,
		Source: source,
	}

	return e.AddNode(name, node)
}

// AddNode adds an arbitrary [TreeNode] as children. The caller is responsible
// for using only valid [Type]s and according fields.
func (e *TreeNode) AddNode(name string, node *TreeNode) (*TreeNode, error) {
	if !e.IsDir() {
		return nil, ErrNodeNotDir
	}

	if ee, exists := e.children[name]; exists {
		return ee, ErrNodeExists
	}

	if e.children == nil {
		e.children = make(map[string]*TreeNode)
	}

	e.children[name] = node

	return node, nil
}

// GetNode gets an [TreeNode] for the given name. Return ErrNodeNotExists if
// it doesn't exist.
func (e *TreeNode) GetNode(name string) (*TreeNode, error) {
	if !e.IsDir() {
		return nil, ErrNodeNotDir
	}

	node, exists := e.children[name]
	if !exists {
		return nil, ErrNodeNotExists
	}

	return node, nil
}

func (e *TreeNode) walk(base string, fn WalkFunc) error {
	for name, node := range e.children {
		path := filepath.Join(base, name)
		if err := fn(path, node); err != nil {
			return err
		}

		if node.IsDir() {
			if err := node.walk(path, fn); err != nil {
				return err
			}
		}
	}

	return nil
}
