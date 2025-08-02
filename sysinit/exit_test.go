// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit_test

import (
	"bytes"
	"testing"

	"github.com/aibor/virtrun/internal/exitcode"
	"github.com/aibor/virtrun/sysinit"
	"github.com/stretchr/testify/assert"
)

func TestPrintExitCode(t *testing.T) {
	var actualOut bytes.Buffer

	sysinit.PrintExitCode(&actualOut, 42)

	assert.Equal(t, exitcode.Sprint(42)+"\n", actualOut.String())
}
