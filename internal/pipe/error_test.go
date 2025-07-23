// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package pipe_test

import (
	"testing"

	"github.com/aibor/virtrun/internal/pipe"
	"github.com/stretchr/testify/assert"
)

func TestError_Is(t *testing.T) {
	//nolint:testifylint
	assert.ErrorIs(t, error(&pipe.Error{}), &pipe.Error{})
	assert.NotErrorIs(t, assert.AnError, &pipe.Error{})
}
