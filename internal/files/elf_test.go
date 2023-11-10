package files_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aibor/virtrun/internal/files"
)

func TestLinkedLibs(t *testing.T) {
	libs, err := files.LinkedLibs("testdata/bin/main")
	require.NoError(t, err)

	expected := []string{
		"libfunc2.so",
		"libfunc3.so",
	}
	assert.Equal(t, expected, libs)
}

func TestELFLibResolverResolve(t *testing.T) {
	defaultSearchPaths := []string{"testdata/lib"}

	tests := []struct {
		name         string
		files        []string
		searchPaths  []string
		expectedLibs []string
		errMsg       string
	}{
		{
			name: "direct reference",
			files: []string{
				"testdata/lib/libfunc3.so",
			},
			searchPaths: defaultSearchPaths,
			expectedLibs: []string{
				"testdata/lib/libfunc1.so",
			},
		},
		{
			name: "indirect reference",
			files: []string{
				"testdata/bin/main",
			},
			searchPaths: defaultSearchPaths,
			expectedLibs: []string{
				"testdata/lib/libfunc2.so",
				"testdata/lib/libfunc3.so",
				// libfunc1.so last since it is referenced indirectly by libfunc3.so.
				"testdata/lib/libfunc1.so",
			},
		},
		{
			name: "libs unique for multiple files",
			files: []string{
				"testdata/bin/main",
				"testdata/lib/libfunc3.so",
			},
			searchPaths: defaultSearchPaths,
			expectedLibs: []string{
				"testdata/lib/libfunc2.so",
				"testdata/lib/libfunc3.so",
				// libfunc1.so last since it is referenced indirectly by libfunc3.so.
				"testdata/lib/libfunc1.so",
			},
		},
		{
			name: "fails if lib not found",
			files: []string{
				"testdata/lib/libfunc3.so",
			},
			errMsg: "lib could not be resolved",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			r := files.ELFLibResolver{
				SearchPaths: tt.searchPaths,
			}

			for _, f := range tt.files {
				err := r.Resolve(f)
				if tt.errMsg == "" {
					assert.NoErrorf(t, err, "must resolve %s", f)
				} else {
					assert.ErrorContains(t, err, tt.errMsg)
				}
			}

			assert.Equal(t, tt.expectedLibs, r.Libs)
		})
	}
}
