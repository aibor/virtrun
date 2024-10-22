// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package initramfs

import (
	"fmt"
	"io/fs"
	"iter"
	"maps"
	"path/filepath"
	"slices"
)

type OpenFunc func(path string) (fs.File, error)

type TreeNamedNode struct {
	*TreeNode
	Name string
}

// TreeNode is a single file tree node.
type TreeNode struct {
	// Type of this node.
	Type TreeNodeType

	// SourceOpenFunc is the content for a virtual regular file.
	SourceOpenFunc OpenFunc

	RelatedPath string

	children map[string]*TreeNode
}

// String returns a string representation of the TreeNode.
func (e *TreeNode) String() string {
	switch e.Type {
	case TreeNodeTypeRegular:
		return "regular file (" + e.RelatedPath + ")"
	case TreeNodeTypeDirectory:
		return fmt.Sprintf("directory (% s)", slices.Collect(maps.Keys(e.children)))
	case TreeNodeTypeLink:
		return "link (" + e.RelatedPath + ")"
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

// AddRegular adds a new regular file [TreeNode] children.
func (e *TreeNode) AddRegular(
	name string,
	path string,
	openFn OpenFunc,
) (*TreeNode, error) {
	node := &TreeNode{
		Type:           TreeNodeTypeRegular,
		SourceOpenFunc: openFn,
		RelatedPath:    path,
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
func (e *TreeNode) AddLink(name, target string) (*TreeNode, error) {
	node := &TreeNode{
		Type:        TreeNodeTypeLink,
		RelatedPath: target,
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
func (e *TreeNode) WriteTo(writer Writer, path string) error {
	switch e.Type {
	case TreeNodeTypeRegular:
		source, err := e.SourceOpenFunc(e.RelatedPath)
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
