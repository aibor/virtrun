// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit_test

import (
	"bytes"
	"testing"

	"github.com/aibor/virtrun/sysinit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFprintError(t *testing.T) {
	var buffer bytes.Buffer

	_, err := sysinit.FprintError(&buffer, assert.AnError)
	require.NoError(t, err)

	expected := "Error: " + assert.AnError.Error() + "\n"
	actual := buffer.String()

	assert.Equal(t, expected, actual)
}

func TestPrinter_PrintWarning(t *testing.T) {
	var buffer bytes.Buffer

	_, err := sysinit.FprintWarning(&buffer, assert.AnError)
	require.NoError(t, err)

	expected := "Warning: " + assert.AnError.Error() + "\n"
	actual := buffer.String()

	assert.Equal(t, expected, actual)
}
