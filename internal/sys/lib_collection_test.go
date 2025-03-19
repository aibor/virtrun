// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sys_test

import (
	"slices"
	"testing"

	"github.com/aibor/virtrun/internal/sys"
	"github.com/stretchr/testify/require"
)

func TestLibCollection_CollectLibsFor(t *testing.T) {
	collection, err := sys.CollectLibsFor(
		t.Context(),
		"testdata/bin/main",
	)
	require.NoError(t, err)

	expectedLibs := []string{
		"testdata/lib/libfunc2.so",
		"testdata/lib/libfunc3.so",
		"testdata/lib/libfunc1.so",
	}

	expectedLinks := []string{
		"testdata/lib",
	}

	actualLibs := slices.Collect(collection.Libs())
	sys.AssertContainsPaths(t, expectedLibs, actualLibs)

	actualLinks := slices.Collect(collection.SearchPaths())
	sys.AssertContainsPaths(t, expectedLinks, actualLinks)
}
