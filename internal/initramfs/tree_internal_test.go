// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package initramfs

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTreeIsRoot(t *testing.T) {
	for _, p := range []string{"", ".", "/", "//"} {
		assert.True(t, isRoot(p), p)
	}
	for _, p := range []string{"-", "_", "\\", "a", "/dir", "/d/"} {
		assert.False(t, isRoot(p), p)
	}
}

func TestTreeGetRoot(t *testing.T) {
	tree := Tree{}
	r := tree.GetRoot()
	assert.NotNil(t, tree.root)
	assert.Equal(t, FileTypeDirectory, tree.root.Type)
	assert.Equal(t, tree.root, r)
}

func TestTreeGetNode(t *testing.T) {
	leafNode := TreeNode{
		Type:        FileTypeRegular,
		RelatedPath: "yo",
	}
	dirNode := TreeNode{
		Type: FileTypeDirectory,
		children: map[string]*TreeNode{
			"leaf": &leafNode,
		},
	}
	tree := Tree{
		root: &TreeNode{
			Type: FileTypeDirectory,
			children: map[string]*TreeNode{
				"dir": &dirNode,
			},
		},
	}

	r, err := tree.GetNode("")
	require.NoError(t, err)
	assert.Equal(t, tree.root, r)

	l, err := tree.GetNode(filepath.Join("dir", "leaf"))
	require.NoError(t, err)
	assert.Equal(t, &leafNode, l)

	d, err := tree.GetNode("/dir")
	require.NoError(t, err)
	assert.Equal(t, &dirNode, d)
}

func TestTreeMkdir(t *testing.T) {
	t.Run("single", func(t *testing.T) {
		tree := Tree{}
		e, err := tree.Mkdir("dir")
		require.NoError(t, err)
		assert.Equal(t, e, tree.GetRoot().children["dir"])
		assert.Equal(t, FileTypeDirectory, e.Type)
		assert.Equal(t, "", e.RelatedPath)
	})

	t.Run("multi", func(t *testing.T) {
		tree := Tree{}
		e, err := tree.Mkdir("sub/dir")
		require.NoError(t, err)
		assert.Equal(t, FileTypeDirectory, e.Type)
		assert.Equal(t, "", e.RelatedPath)
		assert.Empty(t, e.children)
		s, err := tree.GetNode("sub")
		require.NoError(t, err)
		assert.Equal(t, s, tree.GetRoot().children["sub"])
		assert.Equal(t, FileTypeDirectory, s.Type)
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
		assert.Equal(t, FileTypeLink, e.Type)
		assert.Equal(t, "target", e.RelatedPath)
		assert.Empty(t, e.children)
	})

	t.Run("multi", func(t *testing.T) {
		tree := Tree{}
		err := tree.Ln("target", "dir/link")
		require.NoError(t, err)
		e, err := tree.GetNode("dir/link")
		require.NoError(t, err)
		assert.Equal(t, FileTypeLink, e.Type)
		assert.Equal(t, "target", e.RelatedPath)
		assert.Empty(t, e.children)
		s, err := tree.GetNode("dir")
		require.NoError(t, err)
		assert.Equal(t, FileTypeDirectory, s.Type)
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
