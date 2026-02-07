// SPDX-FileCopyrightText: 2026 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sys_test

import (
	"testing"

	"github.com/aibor/virtrun/internal/sys"
	"github.com/stretchr/testify/assert"
)

func TestLDDExecErrorIs(t *testing.T) {
	//nolint:testifylint
	assert.ErrorIs(t, error(&sys.LDDExecError{}), &sys.LDDExecError{})
	assert.NotErrorIs(t, assert.AnError, &sys.LDDExecError{})
}
