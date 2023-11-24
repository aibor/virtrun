package initramfs

import (
	"errors"
)

var (
	// ErrNodeNotDir is returned if a tree node is supposed to be a directory
	// but is not.
	ErrNodeNotDir = errors.New("tree node is not a directory")
	// ErrNodeNotExists is returned if a tree node that is looked up does not exist.
	ErrNodeNotExists = errors.New("tree node does not exist")
	// ErrNodeExists is returned if a tree node exists that was not expected.
	ErrNodeExists = errors.New("tree node already exists")
)
