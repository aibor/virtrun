package initramfs

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aibor/virtrun/internal/files"
)

func TestArchiveNew(t *testing.T) {
	archive := New("first")
	assert.Equal(t, os.DirFS("/"), archive.sourceFS)
	entry, err := archive.fileTree.GetEntry("/init")
	require.NoError(t, err)
	assert.Equal(t, "first", entry.RelatedPath)
	assert.Equal(t, files.TypeRegular, entry.Type)
}

func TestArchiveAddFile(t *testing.T) {
	archive := New("first")

	require.NoError(t, archive.AddFile("second", "rel/third"))
	require.NoError(t, archive.AddFile("", "/abs/fourth"))

	expected := map[string]string{
		"second": "rel/third",
		"fourth": "/abs/fourth",
	}

	for file, relPath := range expected {
		path := filepath.Join("files", file)
		e, err := archive.fileTree.GetEntry(path)
		require.NoError(t, err, path)
		assert.Equal(t, files.TypeRegular, e.Type)
		assert.Equal(t, relPath, e.RelatedPath)
	}
}

func TestArchiveAddFiles(t *testing.T) {
	archive := New("first")

	require.NoError(t, archive.AddFiles("second", "rel/third", "/abs/fourth"))
	require.NoError(t, archive.AddFiles("fifth"))
	require.NoError(t, archive.AddFiles())

	expected := map[string]string{
		"second": "second",
		"third":  "rel/third",
		"fourth": "/abs/fourth",
		"fifth":  "fifth",
	}

	for file, relPath := range expected {
		path := filepath.Join("files", file)
		e, err := archive.fileTree.GetEntry(path)
		require.NoError(t, err, path)
		assert.Equal(t, files.TypeRegular, e.Type)
		assert.Equal(t, relPath, e.RelatedPath)
	}
}

func TestArchiveWriteTo(t *testing.T) {
	testFS := fstest.MapFS{
		"input": &fstest.MapFile{},
	}
	testFile, err := testFS.Open("input")
	require.NoError(t, err)

	test := func(entry *files.Entry, w *MockWriter) error {
		i := Archive{sourceFS: testFS}
		_, err := i.fileTree.GetRoot().AddEntry("init", entry)
		require.NoError(t, err)
		return i.writeTo(w)
	}

	t.Run("unknown file type", func(t *testing.T) {
		err := test(&files.Entry{Type: files.Type(99)}, &MockWriter{})
		assert.ErrorContains(t, err, "unknown file type 99")
	})

	t.Run("nonexisting source", func(t *testing.T) {
		entry := &files.Entry{
			Type:        files.TypeRegular,
			RelatedPath: "nonexisting",
		}
		err := test(entry, &MockWriter{})
		assert.ErrorContains(t, err, "open nonexisting: file does not exist")
	})

	t.Run("existing files", func(t *testing.T) {
		tests := []struct {
			name  string
			entry files.Entry
			mock  MockWriter
		}{
			{
				name: "regular",
				entry: files.Entry{
					Type:        files.TypeRegular,
					RelatedPath: "/input",
				},
				mock: MockWriter{
					Path:   "/init",
					Source: testFile,
					Mode:   0755,
				},
			},
			{
				name: "directory",
				entry: files.Entry{
					Type: files.TypeDirectory,
				},
				mock: MockWriter{
					Path: "/init",
				},
			},
			{
				name: "link",
				entry: files.Entry{
					Type:        files.TypeLink,
					RelatedPath: "/lib",
				},
				mock: MockWriter{
					Path:        "/init",
					RelatedPath: "/lib",
				},
			},
			{
				name: "virtual",
				entry: files.Entry{
					Type:   files.TypeVirtual,
					Source: testFile,
				},
				mock: MockWriter{
					Path:   "/init",
					Source: testFile,
					Mode:   0755,
				},
			},
		}

		for _, tt := range tests {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				t.Run("works", func(t *testing.T) {
					i := Archive{sourceFS: testFS}
					_, err := i.fileTree.GetRoot().AddEntry("init", &tt.entry)
					require.NoError(t, err)
					mock := MockWriter{}
					err = i.writeTo(&mock)
					require.NoError(t, err)
					assert.Equal(t, tt.mock, mock)
				})
				t.Run("fails", func(t *testing.T) {
					i := Archive{sourceFS: testFS}
					_, err := i.fileTree.GetRoot().AddEntry("init", &tt.entry)
					require.NoError(t, err)
					mock := MockWriter{Err: assert.AnError}
					err = i.writeTo(&mock)
					assert.Error(t, err, assert.AnError)
				})
			})
		}
	})
}

func TestArchiveResolveLinkedLibs(t *testing.T) {
	t.Setenv("LD_LIBRARY_PATH", "../files/testdata/lib")
	archive := New("../files/testdata/bin/main")
	err := archive.AddRequiredSharedObjects()
	require.NoError(t, err)

	expectedFiles := map[string]files.Entry{
		"/lib": {
			Type: files.TypeDirectory,
		},
		"/lib/libfunc2.so": {
			Type:        files.TypeRegular,
			RelatedPath: "../files/testdata/lib/libfunc2.so",
		},
		"/lib/libfunc3.so": {
			Type:        files.TypeRegular,
			RelatedPath: "../files/testdata/lib/libfunc3.so",
		},
		"/lib/libfunc1.so": {
			Type:        files.TypeRegular,
			RelatedPath: "../files/testdata/lib/libfunc1.so",
		},
	}

	for f, e := range expectedFiles {
		entry, err := archive.fileTree.GetEntry(f)
		if assert.NoError(t, err, f) {
			assert.Equal(t, e.Type, entry.Type, f)
			if e.RelatedPath != "" {
				expectedPath, err := filepath.Abs(e.RelatedPath)
				require.NoError(t, err)
				assert.Equal(t, expectedPath, entry.RelatedPath, f)
			}
		}
	}
}
