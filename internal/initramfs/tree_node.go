// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package initramfs

import (
	"fmt"
	"io/fs"
	"iter"
	"path/filepath"
	"strings"
)

// TreeNode is a single file tree node.
type TreeNode struct {
	// Type of this node.
	Type TreeNodeType
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
	case TreeNodeTypeRegular:
		return "File from: " + e.RelatedPath
	case TreeNodeTypeDirectory:
		keys := make([]string, 0, len(e.children))
		for key := range e.children {
			keys = append(keys, key)
		}

		return fmt.Sprintf("Dir with entries: % s", keys)
	case TreeNodeTypeLink:
		return "Link to: " + e.RelatedPath
	case TreeNodeTypeVirtual:
		return "File virtual"
	default:
		return "invalid type"
	}
}

// IsDir returns true if the [TreeNode] is a directory.
func (e *TreeNode) IsDir() bool {
	return e.Type == TreeNodeTypeDirectory
}

// IsLink returns true if the [TreeNode] is a link.
func (e *TreeNode) IsLink() bool {
	return e.Type == TreeNodeTypeLink
}

// IsRegular returns true if the [TreeNode] is a regular file.
func (e *TreeNode) IsRegular() bool {
	return e.Type == TreeNodeTypeRegular
}

// IsVirtual returns true if the [TreeNode] is a virtual regular file.
func (e *TreeNode) IsVirtual() bool {
	return e.Type == TreeNodeTypeVirtual
}

// AddRegular adds a new regular file [TreeNode] children.
func (e *TreeNode) AddRegular(name, relatedPath string) (*TreeNode, error) {
	node := &TreeNode{
		Type:        TreeNodeTypeRegular,
		RelatedPath: relatedPath,
	}

	return e.AddNode(name, node)
}

// AddDirectory adds a new directory [TreeNode] children.
func (e *TreeNode) AddDirectory(name string) (*TreeNode, error) {
	node := &TreeNode{
		Type: TreeNodeTypeDirectory,
	}

	return e.AddNode(name, node)
}

// AddLink adds a new link [TreeNode] children.
func (e *TreeNode) AddLink(name, relatedPath string) (*TreeNode, error) {
	node := &TreeNode{
		Type:        TreeNodeTypeLink,
		RelatedPath: relatedPath,
	}

	return e.AddNode(name, node)
}

// AddVirtual adds a new virtual file [TreeNode] children.
func (e *TreeNode) AddVirtual(name string, source fs.File) (*TreeNode, error) {
	node := &TreeNode{
		Type:   TreeNodeTypeVirtual,
		Source: source,
	}

	return e.AddNode(name, node)
}

// AddNode adds an arbitrary [TreeNode] as children. The caller is responsible
// for using only valid [Type]s and according fields.
func (e *TreeNode) AddNode(name string, node *TreeNode) (*TreeNode, error) {
	if !e.IsDir() {
		return nil, ErrTreeNodeNotDir
	}

	if ee, exists := e.children[name]; exists {
		return ee, ErrTreeNodeExists
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
		return nil, ErrTreeNodeNotDir
	}

	node, exists := e.children[name]
	if !exists {
		return nil, ErrTreeNodeNotExists
	}

	return node, nil
}

// WriteTo writes the [TreeNode] into the given [Writer] with the given path.
//
// If the [TreeNode] is a regular file, it is read from the given source
// [fs.FS].
func (e *TreeNode) WriteTo(writer Writer, path string, source fs.FS) error {
	switch e.Type {
	case TreeNodeTypeRegular:
		// Cut leading / since fs.FS considers it invalid.
		relPath := strings.TrimPrefix(e.RelatedPath, "/")

		source, err := source.Open(relPath)
		if err != nil {
			return fmt.Errorf("open source: %w", err)
		}
		defer source.Close()

		//nolint:wrapcheck
		return writer.WriteRegular(path, source, fileMode)
	case TreeNodeTypeDirectory:
		//nolint:wrapcheck
		return writer.WriteDirectory(path)
	case TreeNodeTypeLink:
		//nolint:wrapcheck
		return writer.WriteLink(path, e.RelatedPath)
	case TreeNodeTypeVirtual:
		//nolint:wrapcheck
		return writer.WriteRegular(path, e.Source, fileMode)
	default:
		return fmt.Errorf("%w: %d", ErrTreeNodeTypeUnknown, e.Type)
	}
}

// prefixedPaths creates an iterator over all children.
func (e *TreeNode) prefixedPaths(base string) iter.Seq2[string, *TreeNode] {
	return func(yield func(path string, node *TreeNode) bool) {
		for name, node := range e.children {
			path := filepath.Join(base, name)
			if !yield(path, node) {
				return
			}
		}
	}
}
