// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package initramfs

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTreeIsRoot(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		assert assert.BoolAssertionFunc
	}{
		{
			name:   "empty",
			assert: assert.True,
		},
		{
			name:   "dot",
			path:   ".",
			assert: assert.True,
		},
		{
			name:   "dot with trailing slash",
			path:   "./",
			assert: assert.True,
		},
		{
			name:   "double dot",
			path:   "..",
			assert: assert.True,
		},
		{
			name:   "slash",
			path:   "/",
			assert: assert.True,
		},
		{
			name:   "double slash",
			path:   "//",
			assert: assert.True,
		},
		{
			name:   "hyphen",
			path:   "-",
			assert: assert.False,
		},
		{
			name:   "underscore",
			path:   "_",
			assert: assert.False,
		},
		{
			name:   "backslash",
			path:   "\\",
			assert: assert.False,
		},
		{
			name:   "letter",
			path:   "a",
			assert: assert.False,
		},
		{
			name:   "letter with trailing slash",
			path:   "a/",
			assert: assert.False,
		},
		{
			name:   "subdir",
			path:   "/dir",
			assert: assert.False,
		},
		{
			name:   "subdir with trailing slash",
			path:   "/dir/",
			assert: assert.False,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.assert(t, isRoot(tt.path))
		})
	}
}

func TestTreeGetRoot(t *testing.T) {
	tree := Tree{}
	r := tree.GetRoot()
	assert.NotNil(t, tree.root)
	assert.Equal(t, TreeNodeTypeDirectory, tree.root.Type)
	assert.Equal(t, tree.root, r)
}

func TestTreeGetNode(t *testing.T) {
	leafNode := TreeNode{
		Type:        TreeNodeTypeRegular,
		RelatedPath: "yo",
	}
	dirNode := TreeNode{
		Type: TreeNodeTypeDirectory,
		children: map[string]*TreeNode{
			"leaf": &leafNode,
		},
	}
	tree := Tree{
		root: &TreeNode{
			Type: TreeNodeTypeDirectory,
			children: map[string]*TreeNode{
				"dir": &dirNode,
			},
		},
	}

	tests := []struct {
		name   string
		path   string
		expect *TreeNode
	}{
		{
			name:   "root",
			expect: tree.root,
		},
		{
			name:   "leaf node",
			path:   filepath.Join("dir", "leaf"),
			expect: &leafNode,
		},
		{
			name:   "dir node",
			path:   "/dir",
			expect: &dirNode,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := tree.GetNode(tt.path)
			require.NoError(t, err)
			assert.Equal(t, tt.expect, actual)
		})
	}
}

func TestTreeMkdir(t *testing.T) {
	t.Run("single", func(t *testing.T) {
		tree := Tree{}
		e, err := tree.Mkdir("dir")
		require.NoError(t, err)
		assert.Equal(t, e, tree.GetRoot().children["dir"])
		assert.Equal(t, TreeNodeTypeDirectory, e.Type)
		assert.Equal(t, "", e.RelatedPath)
	})

	t.Run("multi", func(t *testing.T) {
		tree := Tree{}
		e, err := tree.Mkdir("sub/dir")
		require.NoError(t, err)
		assert.Equal(t, TreeNodeTypeDirectory, e.Type)
		assert.Equal(t, "", e.RelatedPath)
		assert.Empty(t, e.children)

		s, err := tree.GetNode("sub")
		require.NoError(t, err)
		assert.Equal(t, s, tree.GetRoot().children["sub"])
		assert.Equal(t, TreeNodeTypeDirectory, s.Type)
		assert.Equal(t, "", s.RelatedPath)
		assert.Equal(t, e, s.children["dir"])
	})

	t.Run("exists", func(t *testing.T) {
		tree := Tree{}
		_, err := tree.Mkdir("dir")
		require.NoError(t, err)
		_, err = tree.Mkdir("dir")
		assert.NoError(t, err)
	})

	t.Run("fails if non-dir exists", func(t *testing.T) {
		tree := Tree{}
		err := tree.Ln("target", "dir")
		require.NoError(t, err)
		_, err = tree.Mkdir("dir")
		assert.Error(t, err)
	})

	t.Run("fails if parent is not dir", func(t *testing.T) {
		tree := Tree{}
		err := tree.Ln("target", "dir")
		require.NoError(t, err)
		_, err = tree.Mkdir("dir/sub")
		assert.Error(t, err)
	})
}

func TestTreeLn(t *testing.T) {
	t.Run("single", func(t *testing.T) {
		tree := Tree{}
		err := tree.Ln("target", "link")
		require.NoError(t, err)
		e, err := tree.GetNode("link")
		require.NoError(t, err)
		assert.Equal(t, e, tree.GetRoot().children["link"])
		assert.Equal(t, TreeNodeTypeLink, e.Type)
		assert.Equal(t, "target", e.RelatedPath)
		assert.Empty(t, e.children)
	})

	t.Run("multi", func(t *testing.T) {
		tree := Tree{}
		err := tree.Ln("target", "dir/link")
		require.NoError(t, err)
		e, err := tree.GetNode("dir/link")
		require.NoError(t, err)
		assert.Equal(t, TreeNodeTypeLink, e.Type)
		assert.Equal(t, "target", e.RelatedPath)
		assert.Empty(t, e.children)

		s, err := tree.GetNode("dir")
		require.NoError(t, err)
		assert.Equal(t, TreeNodeTypeDirectory, s.Type)
		assert.Equal(t, "", s.RelatedPath)
		assert.Equal(t, e, s.children["link"])
	})

	t.Run("exists", func(t *testing.T) {
		tree := Tree{}
		err := tree.Ln("target", "dir/link")
		require.NoError(t, err)
		err = tree.Ln("target", "dir/link")
		assert.NoError(t, err)
	})

	t.Run("fails if non-link exists", func(t *testing.T) {
		tree := Tree{}
		err := tree.Ln("target", "dir/link")
		require.NoError(t, err)
		err = tree.Ln("target", "dir")
		assert.Error(t, err)
	})

	t.Run("fails if parent is not dir", func(t *testing.T) {
		tree := Tree{}
		err := tree.Ln("target", "dir")
		require.NoError(t, err)
		err = tree.Ln("target", "dir/link")
		assert.Error(t, err)
	})
}
