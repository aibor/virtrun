// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package initramfs

import (
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertNode(t *testing.T, i *Initramfs, p string, e TreeNode) {
	t.Helper()

	node, err := i.fileTree.GetNode(p)
	require.NoError(t, err)
	assert.Equal(t, e, *node)
}

func TestInitramfsNew(t *testing.T) {
	testFS := fstest.MapFS{
		"init": &fstest.MapFile{Data: []byte{5, 5}},
	}
	initFile, err := testFS.Open("init")
	require.NoError(t, err, "must open test file")

	tests := []struct {
		name     string
		initFunc func(*TreeNode)
		expected TreeNode
	}{
		{
			name:     "real file",
			initFunc: WithRealInitFile("first"),
			expected: TreeNode{
				Type:        FileTypeRegular,
				RelatedPath: "first",
			},
		},
		{
			name:     "virtual file",
			initFunc: WithVirtualInitFile(initFile),
			expected: TreeNode{
				Type:   FileTypeVirtual,
				Source: initFile,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := New(tt.initFunc)
			assertNode(t, i, "/init", tt.expected)
		})
	}
}

func TestInitramfsAddFile(t *testing.T) {
	archive := New(WithRealInitFile("first"))

	require.NoError(t, archive.AddFile("dir", "second", "rel/third"))
	require.NoError(t, archive.AddFile("dir", "", "/abs/fourth"))

	expected := map[string]string{
		"second": "rel/third",
		"fourth": "/abs/fourth",
	}

	for file, relPath := range expected {
		path := filepath.Join("dir", file)
		e, err := archive.fileTree.GetNode(path)
		require.NoError(t, err, path)
		assert.Equal(t, FileTypeRegular, e.Type)
		assert.Equal(t, relPath, e.RelatedPath)
	}
}

func TestInitramfsAddFiles(t *testing.T) {
	archive := New(WithRealInitFile("first"))

	require.NoError(t, archive.AddFiles("dir", "second", "rel/third", "/abs/fourth"))
	require.NoError(t, archive.AddFiles("dir", "fifth"))
	require.NoError(t, archive.AddFiles("dir"))

	expected := map[string]string{
		"second": "second",
		"third":  "rel/third",
		"fourth": "/abs/fourth",
		"fifth":  "fifth",
	}

	for file, relPath := range expected {
		path := filepath.Join("dir", file)
		e, err := archive.fileTree.GetNode(path)
		require.NoError(t, err, path)
		assert.Equal(t, FileTypeRegular, e.Type)
		assert.Equal(t, relPath, e.RelatedPath)
	}
}

func TestInitramfsWriteTo(t *testing.T) {
	testFS := fstest.MapFS{
		"input": &fstest.MapFile{},
	}
	testFile, err := testFS.Open("input")
	require.NoError(t, err)

	test := func(node *TreeNode, w *MockWriter) error {
		i := Initramfs{}
		_, err := i.fileTree.GetRoot().AddNode("init", node)
		require.NoError(t, err)

		return i.writeTo(w, testFS)
	}

	t.Run("unknown file type", func(t *testing.T) {
		err := test(&TreeNode{Type: FileType(99)}, &MockWriter{})
		assert.ErrorContains(t, err, "unknown file type 99")
	})

	t.Run("nonexisting source", func(t *testing.T) {
		node := &TreeNode{
			Type:        FileTypeRegular,
			RelatedPath: "nonexisting",
		}
		err := test(node, &MockWriter{})
		assert.ErrorContains(t, err, "open nonexisting: file does not exist")
	})

	t.Run("existing files", func(t *testing.T) {
		tests := []struct {
			name string
			node TreeNode
			mock MockWriter
		}{
			{
				name: "regular",
				node: TreeNode{
					Type:        FileTypeRegular,
					RelatedPath: "/input",
				},
				mock: MockWriter{
					Path:   "/init",
					Source: testFile,
					Mode:   0o755,
				},
			},
			{
				name: "directory",
				node: TreeNode{
					Type: FileTypeDirectory,
				},
				mock: MockWriter{
					Path: "/init",
				},
			},
			{
				name: "link",
				node: TreeNode{
					Type:        FileTypeLink,
					RelatedPath: "/lib",
				},
				mock: MockWriter{
					Path:        "/init",
					RelatedPath: "/lib",
				},
			},
			{
				name: "virtual",
				node: TreeNode{
					Type:   FileTypeVirtual,
					Source: testFile,
				},
				mock: MockWriter{
					Path:   "/init",
					Source: testFile,
					Mode:   0o755,
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Run("works", func(t *testing.T) {
					i := Initramfs{}
					_, err := i.fileTree.GetRoot().AddNode("init", &tt.node)
					require.NoError(t, err)

					mock := MockWriter{}
					err = i.writeTo(&mock, testFS)
					require.NoError(t, err)
					assert.Equal(t, tt.mock, mock)
				})
				t.Run("fails", func(t *testing.T) {
					i := Initramfs{}
					_, err := i.fileTree.GetRoot().AddNode("init", &tt.node)
					require.NoError(t, err)

					mock := MockWriter{Err: assert.AnError}
					err = i.writeTo(&mock, testFS)
					require.ErrorIs(t, err, assert.AnError)
				})
			})
		}
	})
}

func TestInitramfsResolveLinkedLibs(t *testing.T) {
	t.Setenv("LD_LIBRARY_PATH", "testdata/lib")

	irfs := New(WithRealInitFile("testdata/bin/main"))
	err := irfs.AddRequiredSharedObjects("")
	require.NoError(t, err)

	expectedFiles := map[string]TreeNode{
		"/lib": {
			Type: FileTypeDirectory,
		},
		"/lib/libfunc2.so": {
			Type:        FileTypeRegular,
			RelatedPath: "testdata/lib/libfunc2.so",
		},
		"/lib/libfunc3.so": {
			Type:        FileTypeRegular,
			RelatedPath: "testdata/lib/libfunc3.so",
		},
		"/lib/libfunc1.so": {
			Type:        FileTypeRegular,
			RelatedPath: "testdata/lib/libfunc1.so",
		},
	}

	for f, e := range expectedFiles {
		node, err := irfs.fileTree.GetNode(f)
		if assert.NoError(t, err, f) {
			assert.Equal(t, e.Type, node.Type, f)

			if e.RelatedPath != "" {
				expectedPath, err := filepath.Abs(e.RelatedPath)
				require.NoError(t, err)
				assert.Equal(t, expectedPath, node.RelatedPath, f)
			}
		}
	}
}
