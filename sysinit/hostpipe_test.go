// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit_test

import (
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aibor/virtrun/sysinit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateNamedPipe(t *testing.T) {
	input := "just\nsome data\n"

	path := filepath.Join(t.TempDir(), "fifo")
	reader, writer, err := sysinit.CreateNamedPipe(path)
	require.NoError(t, err)
	assert.FileExists(t, path)

	go func() {
		defer writer.Close()

		_, _ = io.Copy(writer, strings.NewReader(input))
	}()

	output, err := io.ReadAll(reader)
	require.NoError(t, err)

	assert.Equal(t, input, string(output))
}
