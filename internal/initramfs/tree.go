// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package initramfs

import (
	"errors"
	"iter"
	"path/filepath"
	"strings"
)

// Tree represents a simple file tree.
type Tree struct {
	// Do not access directly! Always use [Tree.GetRoot] to access the root
	// node to ensure it exists.
	root *TreeNode
}

func isRoot(path string) bool {
	switch path {
	case "", ".", "..", string(filepath.Separator):
		return true
	default:
		return false
	}
}

// GetRoot returns the root node of the tree.
func (t *Tree) GetRoot() *TreeNode {
	if t.root == nil {
		t.root = &TreeNode{
			Type: TreeNodeTypeDirectory,
		}
	}

	return t.root
}

func (t *Tree) Nodes(path string) iter.Seq[string] {
	return func(yield func(string) bool) {
		relPath := strings.TrimPrefix(path, "/")
		cleaned := filepath.Clean(relPath)

		if isRoot(cleaned) {
			return
		}

		nodes := strings.Split(cleaned, string(filepath.Separator))
		for _, name := range nodes {
			if !yield(name) {
				return
			}
		}
	}
}

// GetNode returns the node for the given path. Returns ErrNodeNotExists if
// the node does not exist.
func (t *Tree) GetNode(path string) (*TreeNode, error) {
	node := t.GetRoot()

	for name := range t.Nodes(path) {
		var err error

		node, err = node.GetNode(name)
		if err != nil {
			return nil, err
		}
	}

	return node, nil
}

// Mkdir adds a directory node for the given path. Non existing parents
// are created recursively. If any of the parents exists but is not a directory
// ErrNodeNotDir is returned.
func (t *Tree) Mkdir(path string) (*TreeNode, error) {
	node := t.GetRoot()

	for name := range t.Nodes(path) {
		var err error

		node, err = node.AddDirectory(name)
		if err != nil && (!errors.Is(err, ErrTreeNodeExists) || !node.IsDir()) {
			return nil, err
		}
	}

	return node, nil
}

// Ln adds links to target for the given path.
func (t *Tree) Ln(target string, path string) error {
	cleaned := filepath.Clean(path)
	dir, name := filepath.Split(cleaned)

	dirNode, err := t.Mkdir(dir)
	if err != nil {
		return err
	}

	if l, err := dirNode.AddLink(name, target); err != nil {
		if !errors.Is(err, ErrTreeNodeExists) || !l.IsLink() {
			return err
		}
	}

	return nil
}

// All returns an iterator that iterates all [TreeNode]s recursively.
func (t *Tree) All() iter.Seq2[string, *TreeNode] {
	return func(yield func(string, *TreeNode) bool) {
		base := string(filepath.Separator)

		if !yield(base, t.root) {
			return
		}

		// Collect iterators for each sub directory. Start with root directory.
		iterators := []iter.Seq2[string, *TreeNode]{
			t.root.prefixedPaths(base),
		}

		for len(iterators) > 0 {
			for path, node := range iterators[0] {
				if !yield(path, node) {
					return
				}

				if node.IsDir() {
					iterators = append(iterators, node.prefixedPaths(path))
				}
			}

			iterators = iterators[1:]
		}
	}
}
