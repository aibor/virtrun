// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package initramfs

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsRegular(t *testing.T) {
	fileNode := TreeNode{Type: FileTypeRegular}
	dirNode := TreeNode{Type: FileTypeDirectory}
	linkNode := TreeNode{Type: FileTypeLink}
	virtualNode := TreeNode{Type: FileTypeVirtual}

	assert.True(t, fileNode.IsRegular())
	assert.False(t, dirNode.IsRegular())
	assert.False(t, linkNode.IsRegular())
	assert.False(t, virtualNode.IsRegular())
}

func TestIsDir(t *testing.T) {
	fileNode := TreeNode{Type: FileTypeRegular}
	dirNode := TreeNode{Type: FileTypeDirectory}
	linkNode := TreeNode{Type: FileTypeLink}
	virtualNode := TreeNode{Type: FileTypeVirtual}

	assert.False(t, fileNode.IsDir())
	assert.True(t, dirNode.IsDir())
	assert.False(t, linkNode.IsDir())
	assert.False(t, virtualNode.IsDir())
}

func TestIsLink(t *testing.T) {
	fileNode := TreeNode{Type: FileTypeRegular}
	dirNode := TreeNode{Type: FileTypeDirectory}
	linkNode := TreeNode{Type: FileTypeLink}
	virtualNode := TreeNode{Type: FileTypeVirtual}

	assert.False(t, fileNode.IsLink())
	assert.False(t, dirNode.IsLink())
	assert.True(t, linkNode.IsLink())
	assert.False(t, virtualNode.IsLink())
}

func TestIsVirtual(t *testing.T) {
	fileNode := TreeNode{Type: FileTypeRegular}
	dirNode := TreeNode{Type: FileTypeDirectory}
	linkNode := TreeNode{Type: FileTypeLink}
	virtualNode := TreeNode{Type: FileTypeVirtual}

	assert.False(t, fileNode.IsVirtual())
	assert.False(t, dirNode.IsVirtual())
	assert.False(t, linkNode.IsVirtual())
	assert.True(t, virtualNode.IsVirtual())
}

func TestAddFile(t *testing.T) {
	p := TreeNode{Type: FileTypeDirectory}
	e, err := p.AddRegular("file", "source")
	require.NoError(t, err)
	assert.Equal(t, FileTypeRegular, e.Type)
	assert.Equal(t, "source", e.RelatedPath)
	assert.Empty(t, e.Source)
	assert.Empty(t, e.children)
}

func TestAddDirectory(t *testing.T) {
	p := TreeNode{Type: FileTypeDirectory}
	e, err := p.AddDirectory("dir")
	require.NoError(t, err)
	assert.Equal(t, FileTypeDirectory, e.Type)
	assert.Equal(t, "", e.RelatedPath)
	assert.Empty(t, e.Source)
	assert.Empty(t, e.children)
}

func TestAddLink(t *testing.T) {
	p := TreeNode{Type: FileTypeDirectory}
	e, err := p.AddLink("link", "target")
	require.NoError(t, err)
	assert.Equal(t, FileTypeLink, e.Type)
	assert.Equal(t, "target", e.RelatedPath)
	assert.Empty(t, e.Source)
	assert.Empty(t, e.children)
}

func TestAddVirtual(t *testing.T) {
	mapFS := fstest.MapFS{
		"source": &fstest.MapFile{},
	}
	source, err := mapFS.Open("source")
	require.NoError(t, err)

	p := TreeNode{Type: FileTypeDirectory}
	e, err := p.AddVirtual("file", source)
	require.NoError(t, err)
	assert.Equal(t, FileTypeVirtual, e.Type)
	assert.Equal(t, source, e.Source)
	assert.Empty(t, e.RelatedPath)
	assert.Empty(t, e.children)
}

func TestAddNode(t *testing.T) {
	t.Run("new", func(t *testing.T) {
		p := TreeNode{Type: FileTypeDirectory}
		n := TreeNode{}
		e, err := p.AddNode("new", &n)
		require.NoError(t, err)
		assert.Equal(t, &n, e)
	})

	t.Run("exists", func(t *testing.T) {
		p := TreeNode{Type: FileTypeDirectory}
		n := TreeNode{}
		_, err := p.AddNode("new", &n)
		require.NoError(t, err)
		e, err := p.AddNode("new", &n)
		require.ErrorIs(t, err, ErrNodeExists)
		assert.Equal(t, &n, e)
	})

	t.Run("not dir", func(t *testing.T) {
		p := TreeNode{Type: FileTypeRegular}
		n := TreeNode{}
		_, err := p.AddNode("new", &n)
		require.ErrorIs(t, err, ErrNodeNotDir)
	})
}

func TestGetNode(t *testing.T) {
	node := TreeNode{
		Type:        FileTypeRegular,
		RelatedPath: "source",
	}
	p := TreeNode{
		Type: FileTypeDirectory,
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
		assert.ErrorIs(t, err, ErrNodeNotExists)
	})

	t.Run("not dir", func(t *testing.T) {
		p := TreeNode{Type: FileTypeRegular}
		_, err := p.GetNode("file")
		require.ErrorIs(t, err, ErrNodeNotDir)
	})
}
