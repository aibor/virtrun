// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package initramfs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsRegular(t *testing.T) {
	fileNode := TreeNode{Type: TreeNodeTypeRegular}
	dirNode := TreeNode{Type: TreeNodeTypeDirectory}
	linkNode := TreeNode{Type: TreeNodeTypeLink}

	assert.True(t, fileNode.IsRegular())
	assert.False(t, dirNode.IsRegular())
	assert.False(t, linkNode.IsRegular())
}

func TestIsDir(t *testing.T) {
	fileNode := TreeNode{Type: TreeNodeTypeRegular}
	dirNode := TreeNode{Type: TreeNodeTypeDirectory}
	linkNode := TreeNode{Type: TreeNodeTypeLink}

	assert.False(t, fileNode.IsDir())
	assert.True(t, dirNode.IsDir())
	assert.False(t, linkNode.IsDir())
}

func TestIsLink(t *testing.T) {
	fileNode := TreeNode{Type: TreeNodeTypeRegular}
	dirNode := TreeNode{Type: TreeNodeTypeDirectory}
	linkNode := TreeNode{Type: TreeNodeTypeLink}

	assert.False(t, fileNode.IsLink())
	assert.False(t, dirNode.IsLink())
	assert.True(t, linkNode.IsLink())
}

func TestAddFile(t *testing.T) {
	p := TreeNode{Type: TreeNodeTypeDirectory}
	e, err := p.AddRegular("file", "source", OSFileOpen)
	require.NoError(t, err)
	assert.Equal(t, TreeNodeTypeRegular, e.Type)
	assert.Equal(t, "source", e.RelatedPath)
	assert.NotNil(t, e.SourceOpenFunc)
	assert.Empty(t, e.children)
}

func TestAddDirectory(t *testing.T) {
	p := TreeNode{Type: TreeNodeTypeDirectory}
	e, err := p.AddDirectory("dir")
	require.NoError(t, err)
	assert.Equal(t, TreeNodeTypeDirectory, e.Type)
	assert.Nil(t, e.SourceOpenFunc)
	assert.Empty(t, e.RelatedPath)
	assert.Empty(t, e.children)
}

func TestAddLink(t *testing.T) {
	p := TreeNode{Type: TreeNodeTypeDirectory}
	e, err := p.AddLink("link", "target")
	require.NoError(t, err)
	assert.Equal(t, TreeNodeTypeLink, e.Type)
	assert.Nil(t, e.SourceOpenFunc)
	assert.Equal(t, "target", e.RelatedPath)
	assert.Empty(t, e.children)
}

func TestAddNode(t *testing.T) {
	t.Run("new", func(t *testing.T) {
		p := TreeNode{Type: TreeNodeTypeDirectory}
		n := TreeNode{}
		e, err := p.AddNode("new", &n)
		require.NoError(t, err)
		assert.Equal(t, &n, e)
	})

	t.Run("exists", func(t *testing.T) {
		p := TreeNode{Type: TreeNodeTypeDirectory}
		n := TreeNode{}
		_, err := p.AddNode("new", &n)
		require.NoError(t, err)
		e, err := p.AddNode("new", &n)
		require.ErrorIs(t, err, ErrTreeNodeExists)
		assert.Equal(t, &n, e)
	})

	t.Run("not dir", func(t *testing.T) {
		p := TreeNode{Type: TreeNodeTypeRegular}
		n := TreeNode{}
		_, err := p.AddNode("new", &n)
		require.ErrorIs(t, err, ErrTreeNodeNotDir)
	})
}

func TestGetNode(t *testing.T) {
	node := TreeNode{
		Type: TreeNodeTypeRegular,
	}
	p := TreeNode{
		Type: TreeNodeTypeDirectory,
		children: map[string]*TreeNode{
			"file": &node,
		},
	}

	t.Run("exists", func(t *testing.T) {
		e, err := p.GetNode("file")
		require.NoError(t, err)
		assert.Equal(t, &node, e)
	})

	t.Run("does not exist", func(t *testing.T) {
		_, err := p.GetNode("404")
		assert.ErrorIs(t, err, ErrTreeNodeNotExists)
	})

	t.Run("not dir", func(t *testing.T) {
		p := TreeNode{Type: TreeNodeTypeRegular}
		_, err := p.GetNode("file")
		require.ErrorIs(t, err, ErrTreeNodeNotDir)
	})
}
