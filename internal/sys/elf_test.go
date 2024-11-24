// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sys_test

import (
	"testing"

	"github.com/aibor/virtrun/internal/sys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadArch(t *testing.T) {
	arch, err := sys.ReadELFArch("testdata/bin/main")
	require.NoError(t, err)

	assert.Equal(t, sys.Native, arch)
}
