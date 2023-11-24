package initramfs

import (
	"fmt"
	"path/filepath"
)

// Tree represents a simple file tree.
type Tree struct {
	// Do not access directly! Always use [Tree.GetRoot] to access the root
	// node to ensure it exists.
	root *TreeNode
}

func isRoot(path string) bool {
	switch filepath.Clean(path) {
	case "", ".", string(filepath.Separator):
		return true
	default:
		return false
	}
}

// GetRoot returns the root node of the tree.
func (t *Tree) GetRoot() *TreeNode {
	if t.root == nil {
		t.root = &TreeNode{
			Type: FileTypeDirectory,
		}
	}
	return t.root
}

// GetNode returns the node for the given path. Returns ErrNodeNotExists if
// the node does not exist.
func (t *Tree) GetNode(path string) (*TreeNode, error) {
	if isRoot(path) {
		return t.GetRoot(), nil
	}
	dir, name := filepath.Split(filepath.Clean(path))
	parent, err := t.GetNode(dir)
	if err != nil {
		return nil, err
	}
	return parent.GetNode(name)
}

// Mkdir adds a directory node for the given path. Non existing parents
// are created recursively. If any of the parents exists but is not a directory
// ErrNodeNotDir is returned.
func (t *Tree) Mkdir(path string) (*TreeNode, error) {
	cleaned := filepath.Clean(path)
	if isRoot(cleaned) {
		return t.GetRoot(), nil
	}
	dir, name := filepath.Split(cleaned)
	parent, err := t.Mkdir(dir)
	if err != nil {
		return nil, fmt.Errorf("mkdir %s: %v", dir, err)
	}
	node, err := parent.AddDirectory(name)
	if err == ErrNodeExists && node.IsDir() {
		err = nil
	}
	return node, err
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
		if err != ErrNodeExists || !l.IsLink() {
			return err
		}
	}
	return nil
}

// WalkFunc is called with the absolute path to the node.
type WalkFunc func(path string, node *TreeNode) error

// Walk walks the tree recursively, starting at the root, and runs the given
// function for each node. If the function returns an error, the recursion is
// terminated immediately and the error is returned.
func (f *Tree) Walk(fn WalkFunc) error {
	return f.GetRoot().walk(string(filepath.Separator), fn)
}
