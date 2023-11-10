package files

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
	assert.Equal(t, TypeDirectory, tree.root.Type)
	assert.Equal(t, tree.root, r)
}

func TestTreeGetEntry(t *testing.T) {
	leafEntry := Entry{
		Type:        TypeRegular,
		RelatedPath: "yo",
	}
	dirEntry := Entry{
		Type: TypeDirectory,
		children: map[string]*Entry{
			"leaf": &leafEntry,
		},
	}
	tree := Tree{
		root: &Entry{
			Type: TypeDirectory,
			children: map[string]*Entry{
				"dir": &dirEntry,
			},
		},
	}

	r, err := tree.GetEntry("")
	require.NoError(t, err)
	assert.Equal(t, tree.root, r)

	l, err := tree.GetEntry(filepath.Join("dir", "leaf"))
	require.NoError(t, err)
	assert.Equal(t, &leafEntry, l)

	d, err := tree.GetEntry("/dir")
	require.NoError(t, err)
	assert.Equal(t, &dirEntry, d)
}

func TestTreeMkdir(t *testing.T) {
	t.Run("single", func(t *testing.T) {
		tree := Tree{}
		e, err := tree.Mkdir("dir")
		require.NoError(t, err)
		assert.Equal(t, e, tree.GetRoot().children["dir"])
		assert.Equal(t, TypeDirectory, e.Type)
		assert.Equal(t, "", e.RelatedPath)
	})

	t.Run("multi", func(t *testing.T) {
		tree := Tree{}
		e, err := tree.Mkdir("sub/dir")
		require.NoError(t, err)
		assert.Equal(t, TypeDirectory, e.Type)
		assert.Equal(t, "", e.RelatedPath)
		assert.Empty(t, e.children)
		s, err := tree.GetEntry("sub")
		require.NoError(t, err)
		assert.Equal(t, s, tree.GetRoot().children["sub"])
		assert.Equal(t, TypeDirectory, s.Type)
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
		e, err := tree.GetEntry("link")
		require.NoError(t, err)
		assert.Equal(t, e, tree.GetRoot().children["link"])
		assert.Equal(t, TypeLink, e.Type)
		assert.Equal(t, "target", e.RelatedPath)
		assert.Empty(t, e.children)
	})

	t.Run("multi", func(t *testing.T) {
		tree := Tree{}
		err := tree.Ln("target", "dir/link")
		require.NoError(t, err)
		e, err := tree.GetEntry("dir/link")
		require.NoError(t, err)
		assert.Equal(t, TypeLink, e.Type)
		assert.Equal(t, "target", e.RelatedPath)
		assert.Empty(t, e.children)
		s, err := tree.GetEntry("dir")
		require.NoError(t, err)
		assert.Equal(t, TypeDirectory, s.Type)
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
